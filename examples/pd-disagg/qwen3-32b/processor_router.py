from components.frontend import Frontend
from components.kv_router import Router
from components.processor import Processor

Frontend.link(Processor).link(Router)