from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    cors_origins: list[str] = ["http://localhost:1420"]

    openrouter_api_key: str | None = None
    google_api_key: str | None = None


settings = Settings()
