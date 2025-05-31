import time
import json
import base64

import grpc
from concurrent import futures
import api_pb2
import api_pb2_grpc

import pika

import analysis

RABBITMQ_URL = "amqp://guest:guest@0.0.0.0:5672/"
RABBITMQ_QUEUE = "analysis_queue"

class AnalyzerServicer(api_pb2_grpc.AnalyzerServiceServicer):
    def __init__(self, rabbit_channel):
        self.ch = rabbit_channel
    def Analyze(self, request_iterator, context):
        for audio_buf in request_iterator:
            sensor_id = audio_buf.sensor_id
            coeff, lag = analysis.calculate_correlation_coefficient(audio_buf.pcm)

            wav_bytes = analysis.buffer_to_wav_bytes(pcm_bytes=audio_buf.pcm, sample_rate=16000)
            wav_b64 = base64.b64encode(wav_bytes).decode("ascii")

            msg_dict = {
                "sensor_id":  sensor_id,
                "coeff":      coeff,
                "lag":        lag,
                "wav_buffer": wav_b64,
            }
            msg_json = json.dumps(msg_dict)

            try:
                self.ch.basic_publish(
                    exchange="",
                    routing_key=RABBITMQ_QUEUE,
                    body=msg_json.encode("utf-8"),
                    properties=pika.BasicProperties(
                        delivery_mode=2
                    )
                )
                print(f"[RabbitMQ] Published analysis for sensor {sensor_id}: coeff={coeff:.4f}, lag={lag:.4f}")
            except Exception as e:
                # If RabbitMQ goes down or errors, we just log it
                print(f"[ERROR] Failed to publish to RabbitMQ: {e}")

        return api_pb2.Empty()

def serve():
    connection = pika.BlockingConnection(pika.URLParameters(RABBITMQ_URL))
    channel = connection.channel()
    channel.queue_declare(queue=RABBITMQ_QUEUE, durable=True)
    print(f"[RabbitMQ] Connected and declared queue '{RABBITMQ_QUEUE}'")

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    api_pb2_grpc.add_AnalyzerServiceServicer_to_server(AnalyzerServicer(rabbit_channel=channel), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    print("Python gRPC server listening on 0.0.0.0:50051")
    try:
        while True:
            time.sleep(86400)
    except KeyboardInterrupt:
        server.stop(0)

if __name__ == '__main__':
    serve()
