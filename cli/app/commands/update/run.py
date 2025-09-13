import subprocess

from app.commands.service.base import BaseDockerCommandBuilder, BaseDockerService
from app.utils.logger import Logger
from app.utils.config import Config, DEFAULT_COMPOSE_FILE
from app.utils.config import NIXOPUS_CONFIG_DIR
from .messages import (
    updating_nixopus,
    pulling_latest_images,
    images_pulled_successfully,
    failed_to_pull_images,
    starting_services,
    failed_to_start_services,
    nixopus_updated_successfully,
)

# TODO: Add support for staging or forked repository update is not availlable yet
class Update:
    def __init__(self, logger: Logger):
        self.logger = logger
        self.config = Config()

    def run(self):
        compose_file = self.config.get_yaml_value(DEFAULT_COMPOSE_FILE)
        compose_file_path = self.config.get_yaml_value(NIXOPUS_CONFIG_DIR) + "/" + compose_file
        self.logger.info(updating_nixopus)
        
        docker_service = BaseDockerService(self.logger, "pull")
        
        self.logger.debug(pulling_latest_images)
        success, output = docker_service.execute_services(compose_file=compose_file_path)
        
        if not success:
            self.logger.error(failed_to_pull_images.format(error=output))
            return
        
        self.logger.debug(images_pulled_successfully)
        
        docker_service_up = BaseDockerService(self.logger, "up")
        self.logger.debug(starting_services)
        success, output = docker_service_up.execute_services(compose_file=compose_file_path, detach=True)
        
        if not success:
            self.logger.error(failed_to_start_services.format(error=output))
            return
        
        self.logger.info(nixopus_updated_successfully)
