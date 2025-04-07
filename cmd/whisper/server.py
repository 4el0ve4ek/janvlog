import whisper_timestamped as whisper

import uvicorn
from fastapi import FastAPI

model = whisper.load_model("turbo")

app = FastAPI()
@app.post("/transcribe")
async def transcribe(audio: str):
    return whisper.transcribe_timestamped(model, audio, language='ru', vad=True, temperature=(0.0, 0.2, 0.4, 0.6, 0.8, 1.0), detect_disfluencies=True)

uvicorn.run(app, host="::", port=8080)
