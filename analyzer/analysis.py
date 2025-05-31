import io
import numpy as np
import wave
import librosa


def buffer_to_wav_bytes(pcm_bytes: bytes, sample_rate=44100, channels=1, bits=16) -> bytes:
    buf = io.BytesIO()
    with wave.open(buf, 'wb') as w:
        w.setnchannels(channels)
        w.setsampwidth(bits // 8)
        w.setframerate(sample_rate)
        w.writeframes(pcm_bytes)
    return buf.getvalue()

def calculate_correlation_coefficient(
    rec_pcm_bytes: bytes,
    sample_rate: int = 16000,
    channels: int = 1,
    bits: int = 16
) -> tuple[float, int]:
    wav_rec_bytes = buffer_to_wav_bytes(rec_pcm_bytes, sample_rate, channels, bits)
    rec_buffer = io.BytesIO(wav_rec_bytes)
    rec, sr_rec = librosa.load(rec_buffer, sr=sample_rate, mono=True)

    ref, sr_ref = librosa.load('ref.wav', sr=sample_rate, mono=True)

    if sr_rec != sr_ref:
        raise ValueError(f"Frequency mismatch: rec={sr_rec}, ref={sr_ref}")

    rec = rec / np.max(np.abs(rec)) if np.max(np.abs(rec)) > 0 else rec
    ref = ref / np.max(np.abs(ref)) if np.max(np.abs(ref)) > 0 else ref

    corr_full = np.correlate(rec, ref, mode='full')

    idx_max = np.argmax(np.abs(corr_full))
    lag = idx_max - (len(ref) - 1)

    norm_rec = np.linalg.norm(rec)
    norm_ref = np.linalg.norm(ref)
    if norm_rec == 0 or norm_ref == 0:
        corr_coeff = 0.0
    else:
        corr_coeff = corr_full[idx_max] / (norm_rec * norm_ref)

    return corr_coeff.__float__(), lag.__int__()
