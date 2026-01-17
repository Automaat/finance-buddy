from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    database_url: str
    app_password: str
    cors_origins: str
    owner_names: str = "Marcin,Ewa,Shared"
    default_owner: str = "Marcin"

    model_config = SettingsConfigDict(env_file=".env")

    @property
    def owner_names_list(self) -> list[str]:
        """Parse comma-separated owner names into a list"""
        return [name.strip() for name in self.owner_names.split(",")]


settings = Settings()
