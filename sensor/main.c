// sensor.c
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include <strings.h>
#include <unistd.h>
#include <alsa/asoundlib.h>
#include <mosquitto.h>
#include <signal.h>

#define MQTT_BROKER   "0.0.0.0"
#define MQTT_PORT     1883
#define MQTT_TOPIC    "sensors/audio/%d"
#define MQTT_TOPIC_STATUS    "sensors/status/%d"

#define SAMPLE_RATE     16000
#define CHANNELS        1
#define FRAMES_PER_BUF  1024


#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <arpa/inet.h>
#include <alsa/asoundlib.h>

struct mosquitto *mosq;
snd_pcm_t *pcm;
int sensor_id;

void publish_status(char* status) {
	char topic[50];
	snprintf(topic, sizeof(topic), MQTT_TOPIC_STATUS, sensor_id);
	mosquitto_publish(mosq, NULL, topic,
			strlen(status), status, 1, false);
}

void sig_handler(int signo) {
	if (signo == SIGINT) {
		publish_status("offline");

		mosquitto_loop_stop(mosq, 1);
		mosquitto_destroy(mosq);
		mosquitto_lib_cleanup();
		snd_pcm_close(pcm);

		exit(0);
	}
}

int main(int argc, char *argv[]) {
	char *p;
	sensor_id = (int)strtol(argv[1], &p, 10);
	printf("Initialized Sensor %d\n", sensor_id);

	int err;
	snd_pcm_hw_params_t *hw;
	uint32_t seq = 0;
	int16_t buffer[FRAMES_PER_BUF];

	if ((err = snd_pcm_open(&pcm, "default", SND_PCM_STREAM_CAPTURE, 0)) < 0) {
		fprintf(stderr, "ALSA open error: %s\n", snd_strerror(err));
		return 1;
	}
	snd_pcm_hw_params_malloc(&hw);
	snd_pcm_hw_params_any(pcm, hw);
	snd_pcm_hw_params_set_access(pcm, hw, SND_PCM_ACCESS_RW_INTERLEAVED);
	snd_pcm_hw_params_set_format(pcm, hw, SND_PCM_FORMAT_S16_LE);
	snd_pcm_hw_params_set_channels(pcm, hw, CHANNELS);
	snd_pcm_hw_params_set_rate(pcm, hw, SAMPLE_RATE, 0);
	snd_pcm_hw_params(pcm, hw);
	snd_pcm_prepare(pcm);
	snd_pcm_hw_params_free(hw);

	mosquitto_lib_init();

	char client_id[50];
	sprintf(client_id, "sensor-%d", sensor_id);
	mosq = mosquitto_new(client_id, 1, NULL);
	if (!mosq) {
		fprintf(stderr, "Mosquitto init failed\n");
		return 1;
	}
	if (mosquitto_connect(mosq, MQTT_BROKER, MQTT_PORT, 60) != MOSQ_ERR_SUCCESS) {
		fprintf(stderr, "Mosquitto connect error\n");
		return 1;
	}
	printf("Connected to tcp://%s:%d\n", MQTT_BROKER, MQTT_PORT);
	mosquitto_loop_start(mosq);
	publish_status("online");

	char topic[50];
	snprintf(topic, sizeof(topic), MQTT_TOPIC, sensor_id);
	printf("Topic: %s\n", topic);


	if (signal(SIGINT, sig_handler) == SIG_ERR) {
		 printf("\ncan't catch SIGINT\n");
	}
	while (1) {
		err = snd_pcm_readi(pcm, buffer, FRAMES_PER_BUF);
		if (err < 0) {
			snd_pcm_prepare(pcm);
			continue;
		} else if (err != FRAMES_PER_BUF) {
			fprintf(stderr, "Short read: %d frames\n", err);
		}

		// Build payload: 4-byte seq + PCM data
		size_t payload_len = sizeof(seq) + sizeof(buffer);
		uint8_t *payload = malloc(payload_len);
		uint32_t be_seq = htonl(seq);
		memcpy(payload, &be_seq, sizeof(be_seq));
		memcpy(payload + sizeof(be_seq), buffer, sizeof(buffer));

		mosquitto_publish(mosq, NULL, topic,
				payload_len, payload, 1, false);

		free(payload);
		seq++;

		// block thread for ~buffer_duration = 1024 / 16000 ≈ 64 ms
		usleep((FRAMES_PER_BUF * 1000000) / SAMPLE_RATE);
	}

	return 0;
}
