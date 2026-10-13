import sys
import threading
import time
import urllib.error
import urllib.request
from collections import Counter

URL = "http://localhost:30080/"


def error_type(e):
    if isinstance(e, urllib.error.HTTPError):
        return "http_" + str(e.code)
    if isinstance(e, urllib.error.URLError):
        e = e.reason
    if isinstance(e, TimeoutError):
        return "timeout"
    if isinstance(e, ConnectionRefusedError):
        return "refused"
    if isinstance(e, ConnectionResetError):
        return "reset"
    return "other"


def one_request(lines, lock):
    t = time.strftime("%H:%M:%S")
    try:
        body = urllib.request.urlopen(URL, timeout=1).read().decode().strip()
        line = t + " ok " + body
    except Exception as e:
        line = t + " error " + error_type(e)
    with lock:
        lines.append(line)


def run_load(seconds, filename):
    lines = []
    lock = threading.Lock()
    threads = []
    end = time.time() + seconds
    while time.time() < end:
        th = threading.Thread(target=one_request, args=(lines, lock))
        th.start()
        threads.append(th)
        time.sleep(0.05)
    for th in threads:
        th.join()
    lines.sort()
    with open(filename, "w") as f:
        f.write("\n".join(lines) + "\n")


def read_log(filename):
    with open(filename) as f:
        return f.read().splitlines()


def stats(filename):
    lines = read_log(filename)
    ok = [l for l in lines if " ok " in l]
    errors = Counter(l.split()[2] for l in lines if " error " in l)
    versions = Counter(l.split("version=")[1].split()[0] for l in ok)
    pods = Counter(l.split("pod=")[1] for l in ok)

    text = "всего запросов: " + str(len(lines)) + "\n"
    text += "успешных: " + str(len(ok)) + "\n"
    text += "ошибок: " + str(sum(errors.values())) + "\n"
    for name, count in sorted(errors.items()):
        text += "  " + name + ": " + str(count) + "\n"
    text += "по версиям:\n"
    for name, count in sorted(versions.items()):
        text += "  " + name + ": " + str(count) + "\n"
    text += "по подам:\n"
    for name, count in sorted(pods.items()):
        text += "  " + name + ": " + str(count) + "\n"
    return text


def per_second(filename, words):
    lines = read_log(filename)
    seconds = sorted(set(l.split()[0] for l in lines))
    text = ""
    for s in seconds:
        in_second = [l for l in lines if l.startswith(s)]
        text += s
        for w in words:
            text += "  " + w + ": " + str(sum(1 for l in in_second if w in l))
        text += "\n"
    return text


if __name__ == "__main__":
    seconds = int(sys.argv[1])
    run_load(seconds, "load.txt")
    print(stats("load.txt"))
