import whisper_timestamped as whisper

import uvicorn
from fastapi import FastAPI

model = whisper.load_model("small")

app = FastAPI()
@app.post("/transcribe")
async def transcribe(audio: str):
    return whisper.transcribe_timestamped(model, audio, language='ru')

uvicorn.run(app, host="::", port=8080)
