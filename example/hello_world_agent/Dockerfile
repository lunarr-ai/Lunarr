FROM python:3.13-slim

WORKDIR /app

COPY requirements.txt .

RUN pip install --no-cache-dir -r requirements.txt

COPY . .

ENV GOOGLE_API_KEY=""

EXPOSE 8001

CMD ["uvicorn", "main:root_agent", "--host", "0.0.0.0", "--port", "8001"]
