from abc import ABC, abstractmethod

class Seeder(ABC):
    @property
    def name(self) -> str:
        return self.__class__.__name__

    @abstractmethod
    def seed(self) -> None:
        pass

class SeederService:
    _seeder: list[Seeder]

    def __init__(self, seeder: list[Seeder]):
        self._seeder = seeder

    def seed(self) -> None:
        for seeder in self._seeder:
            print(f"Seeding: {seeder.name}")
            seeder.seed()
