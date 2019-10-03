from locust import HttpLocust, TaskSet
import locust_prometheus


def login(l):
    l.client.get("https://loadtestsiteexample.com/login")

def index(l):
    l.client.get("https://loadtestsiteexample.com/index")

def profile(l):
    l.client.get("https://loadtestsite.com/profile")

class UserBehavior(TaskSet):
    tasks = {index: 2, profile: 1}

    def on_start(self):
        login(self)

class WebsiteUser(HttpLocust):
    host = 'https://loadtestsiteexample.com'
    task_set = UserBehavior
    min_wait = 5000
    max_wait = 9000