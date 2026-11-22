import os
import subprocess
import threading
import time

from load import per_second, run_load, stats

os.chdir(os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))


def kubectl(*args):
    return subprocess.run(["kubectl", *args], capture_output=True, text=True).stdout


out = open("evidence/scenario2-kubectl.txt", "w")


def log(text):
    print(text)
    out.write(text + "\n")


pod = kubectl("get", "pods", "-l", "app=hello", "-o", "name").split()[0].replace("pod/", "")

load = threading.Thread(target=run_load, args=(40, "evidence/scenario2-load.txt"))
load.start()

time.sleep(10)
log(time.strftime("%H:%M:%S") + " /unready на " + pod)
log(kubectl("exec", pod, "--", "wget", "-qO-", "--post-data=", "localhost:8080/unready"))

time.sleep(6)
log(kubectl("get", "pods"))
log(kubectl("get", "endpoints", "hello"))

time.sleep(10)
log(time.strftime("%H:%M:%S") + " /ready-on на " + pod)
log(kubectl("exec", pod, "--", "wget", "-qO-", "--post-data=", "localhost:8080/ready-on"))

time.sleep(4)
log(kubectl("get", "pods"))
log(kubectl("get", "endpoints", "hello"))
out.close()

load.join()

result = stats("evidence/scenario2-load.txt")
print(result)
open("evidence/scenario2-stats.txt", "w").write(result)
open("evidence/scenario2-traffic.txt", "w").write(per_second("evidence/scenario2-load.txt", [pod]))
