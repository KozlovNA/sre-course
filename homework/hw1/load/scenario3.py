import os
import subprocess
import threading
import time

from load import per_second, run_load, stats

os.chdir(os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))


def kubectl(*args):
    return subprocess.run(["kubectl", *args], capture_output=True, text=True).stdout


out = open("evidence/scenario3-rollout.txt", "w")


def log(text):
    print(text)
    out.write(text + "\n")


kubectl("set", "image", "deployment/hello", "hello=hello-app:v1")
kubectl("rollout", "status", "deployment/hello")
time.sleep(10)

watch = subprocess.Popen(["kubectl", "get", "pods", "-w"], stdout=open("evidence/scenario3-pods.txt", "w"))
load = threading.Thread(target=run_load, args=(50, "evidence/scenario3-load.txt"))
load.start()

time.sleep(10)
log(time.strftime("%H:%M:%S") + " kubectl set image deployment/hello hello=hello-app:v2")
log(kubectl("set", "image", "deployment/hello", "hello=hello-app:v2"))
log(kubectl("rollout", "status", "deployment/hello"))
log(time.strftime("%H:%M:%S") + " rollout done")
log(kubectl("get", "rs"))
out.close()

load.join()
watch.terminate()

result = stats("evidence/scenario3-load.txt")
print(result)
open("evidence/scenario3-stats.txt", "w").write(result)
open("evidence/scenario3-versions.txt", "w").write(per_second("evidence/scenario3-load.txt", ["version=v1", "version=v2"]))
