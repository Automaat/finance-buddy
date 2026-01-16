from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    database_url: str
    app_password: str
    cors_origins: str

    model_config = SettingsConfigDict(env_file=".env")


settings = Settings()
