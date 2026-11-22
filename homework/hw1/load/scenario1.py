import os
import subprocess
import threading
import time

from load import run_load, stats

os.chdir(os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))


def kubectl(*args):
    return subprocess.run(["kubectl", *args], capture_output=True, text=True).stdout


pod = kubectl("get", "pods", "-l", "app=hello", "-o", "name").split()[0]

watch = subprocess.Popen(["kubectl", "get", "pods", "-w"], stdout=open("evidence/scenario1-pods.txt", "w"))
load = threading.Thread(target=run_load, args=(40, "evidence/scenario1-load.txt"))
load.start()

time.sleep(10)
action = time.strftime("%H:%M:%S") + " kubectl delete " + pod
print(action)
open("evidence/scenario1-actions.txt", "w").write(action + "\n")
print(kubectl("delete", pod))

load.join()
watch.terminate()

result = stats("evidence/scenario1-load.txt")
print(result)
open("evidence/scenario1-stats.txt", "w").write(result)
