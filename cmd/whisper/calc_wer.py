import whisper_timestamped as whisper
from datasets import load_dataset, Audio
from jiwer import wer
from tqdm import tqdm
import pandas as pd
import torch
import os
import time
import re

from transformers.models.whisper.english_normalizer import BasicTextNormalizer

dataset = load_dataset("mozilla-foundation/common_voice_17_0", "ru", split="test", token=os.environ["HF_TOKEN"])
dataset = dataset.cast_column("audio", Audio(sampling_rate=16000))

# for faster testing
dataset = dataset.select(range(min(10, len(dataset))))


model = whisper.load_model("turbo")


def transcribe_audio(audio_array, sampling_rate):
    audio = torch.tensor(audio_array).to(torch.float)

    return whisper.transcribe_timestamped(model, audio, language='ru', vad=True, detect_disfluencies=False)["text"]


def calculate_wer(reference, hypothesis):
    if not reference or not hypothesis:
        return 1.0 
    
    reference = simple_ru_normalizer(reference)
    hypothesis = simple_ru_normalizer(hypothesis)
    
    return wer(reference, hypothesis)

def simple_ru_normalizer(text):
    text = text.lower()
    text = re.sub(r'[^а-яёa-z0-9 ]+', ' ', text, flags=re.IGNORECASE)
    text = re.sub(r'\s+', ' ', text).strip()
    return text

results = []

refs = []
hyps = []

for idx, item in enumerate(tqdm(dataset)):
    audio_array = item["audio"]["array"]
    sampling_rate = item["audio"]["sampling_rate"]
    
    reference = item.get("sentence", item.get("text", ""))
    if not reference:
        print(f"Пропуск элемента {idx} - эталонный текст не найден")
        continue
    start_time = time.time()
    hypothesis = transcribe_audio(audio_array, sampling_rate)
    processing_time = time.time() - start_time
    wer_score = calculate_wer(reference, hypothesis)
    refs.append(reference)
    hyps.append(hypothesis)
    
    results.append({
        "reference": reference,
        "hypothesis": hypothesis,
        "wer": wer_score,
        # "duration": item["audio"]["duration"],
        # "processing_time": processing_time,
    })
    
    



results_df = pd.DataFrame(results)

average_wer = results_df["wer"].mean()
print(f"Средний WER: {average_wer:.4f}")
print(results_df)