# ДЗ 1 Kubernetes

Скринкаст: TODO

Таймкоды:
- 00:00 что будет в видео
- TODO манифесты
- TODO запуск
- TODO сценарий 1
- TODO сценарий 2
- TODO сценарий 3
- TODO выводы

## Что сделал

Приложение на Go, отвечает на `/` строкой `hello version=v1 pod=<имя пода>`. Еще есть `/live`, `/ready`, `/unready` и `/ready-on`. На SIGTERM вызывается `server.Shutdown`, он ждет, пока закончатся текущие запросы.

Версия задается при сборке через `--build-arg VERSION=v1`.

Запускал в kind, 2 реплики, Service типа NodePort на порту 30080.

## Версии

- Ubuntu 24.04 (WSL2)
- Docker 29.4.3
- kind v0.33.0
- kubectl v1.37.1, кластер v1.37.0
- Go 1.26

## Схема

```
load.py -> localhost:30080 -> нода kind -> Service hello (NodePort 30080)
                                                     |
                                                     v
                                          Endpoints (только Ready поды)
                                             |                 |
                                             v                 v
                                          Pod 1 :8080       Pod 2 :8080
```

Service выбирает поды по метке `app=hello`. В Endpoints попадают только поды, у которых проходит readinessProbe.

## Как запустить

```bash
kind create cluster --config kubernetes/kind-config.yaml

docker build --build-arg VERSION=v1 -t hello-app:v1 app/
docker build --build-arg VERSION=v2 -t hello-app:v2 app/
kind load docker-image hello-app:v1 hello-app:v2

kubectl apply -f kubernetes/deployment.yaml -f kubernetes/service.yaml
kubectl get pods
curl localhost:30080
```

Результат в [evidence/start.txt](evidence/start.txt).

## Манифесты

- 2 реплики, чтобы когда одна недоступна, работала вторая
- requests 50m/32Mi и limits 200m/64Mi
- readinessProbe на `/ready` раз в 2 секунды, после 2 ошибок под убирается из Endpoints. Сделал часто, чтобы под быстро переставал получать трафик
- livenessProbe на `/live` раз в 10 секунд, после 3 ошибок рестарт. Сделал реже и на другом адресе, чтобы под не перезапускался, когда он просто not-ready
- RollingUpdate с `maxSurge: 1` и `maxUnavailable: 0`: сначала поднимается новый под и только потом удаляется старый, поэтому во время обновления всегда есть 2 рабочих пода
- preStop `sleep 5`: под ждет 5 секунд перед SIGTERM, чтобы его успели убрать из Endpoints, пока он еще отвечает

## Нагрузка

[load/load.py](load/load.py) - каждые 0.05 секунды запускает запрос в отдельном потоке, таймаут 1 секунда. Получается 20 запросов в секунду. Каждый запрос пишется строкой в лог:

```
23:43:30 ok hello version=v1 pod=hello-dd7577b74-dbf5j
23:43:30 error timeout
```

Типы ошибок: `timeout`, `refused` (отказ соединения), `reset` (сброс соединения), `http_500` и т.п. (5xx), `other`. Функция `stats` считает итог: сколько запросов, сколько ошибок каждого типа, сколько ответов от каждой версии и каждого пода.

Можно запустить отдельно: `python3 load/load.py 30` (30 секунд).

## Сценарии

Каждый сценарий - скрипт на Python в `load/`, все логи сохраняются в `evidence/`:

```bash
python3 load/scenario1.py
python3 load/scenario2.py
python3 load/scenario3.py
```

## Сценарий 1. Удаление пода

[load/scenario1.py](load/scenario1.py): запускает нагрузку на 40 секунд и `kubectl get pods -w`, через 10 секунд удаляет первый под.

Результат ([scenario1-stats.txt](evidence/scenario1-stats.txt)):

```
всего запросов: 793
успешных: 793
ошибок: 0
по подам:
  hello-dd7577b74-dbf5j: 92    <- удалил
  hello-dd7577b74-mbvr4: 424
  hello-dd7577b74-qv457: 277   <- новый
```

Что было с подами ([scenario1-pods.txt](evidence/scenario1-pods.txt)):

```
hello-dd7577b74-dbf5j   1/1     Terminating         0          21s
hello-dd7577b74-qv457   0/1     Pending             0          0s
hello-dd7577b74-qv457   0/1     ContainerCreating   0          0s
hello-dd7577b74-qv457   1/1     Running             0          1s
hello-dd7577b74-dbf5j   0/1     Completed           0          26s
```

Удалил под в 23:43:30 ([scenario1-actions.txt](evidence/scenario1-actions.txt)). Сразу создался новый, примерно через секунду он стал Ready (1/1), то есть Ready-подов было 1 из 2 около секунды. Последний ответ от старого пода был в 23:43:30, первый от нового в 23:43:32. Ошибок нет, потому что второй под все это время работал, а старый благодаря preStop не закрылся сразу.

## Сценарий 2. Not-ready

[load/scenario2.py](load/scenario2.py): запускает нагрузку на 40 секунд, через 10 секунд делает `/unready` на первом поде через `kubectl exec` (port-forward нельзя), смотрит поды и Endpoints, через 16 секунд делает `/ready-on`.

После `/unready` ([scenario2-kubectl.txt](evidence/scenario2-kubectl.txt)):

```
NAME                    READY   STATUS    RESTARTS   AGE
hello-dd7577b74-mbvr4   0/1     Running   0          68s
hello-dd7577b74-qv457   1/1     Running   0          46s

NAME    ENDPOINTS          AGE
hello   10.244.0.22:8080   17m
```

Под mbvr4 стал 0/1 и пропал из Endpoints, RESTARTS 0. После `/ready-on` он вернулся:

```
hello-dd7577b74-mbvr4   1/1     Running   0          82s

NAME    ENDPOINTS                           AGE
hello   10.244.0.20:8080,10.244.0.22:8080   18m
```

Сколько запросов приходило на этот под по секундам ([scenario2-traffic.txt](evidence/scenario2-traffic.txt)):

```
23:44:10  hello-dd7577b74-mbvr4: 11   <- /unready
23:44:11  hello-dd7577b74-mbvr4: 11
23:44:12  hello-dd7577b74-mbvr4: 6
23:44:13  hello-dd7577b74-mbvr4: 5
23:44:14  hello-dd7577b74-mbvr4: 0
...
23:44:26  hello-dd7577b74-mbvr4: 0    <- /ready-on
23:44:27  hello-dd7577b74-mbvr4: 5
```

Итог: 793 запроса, 0 ошибок ([scenario2-stats.txt](evidence/scenario2-stats.txt)).

Трафик на под прекратился примерно через 4 секунды после `/unready` (проба раз в 2 секунды, нужно 2 ошибки). Под не перезапустился, потому что liveness проверяет `/live`, а не `/ready`.

## Сценарий 3. Обновление v1 -> v2

[load/scenario3.py](load/scenario3.py): сначала ставит v1, потом запускает нагрузку на 50 секунд и через 10 секунд делает `kubectl set image deployment/hello hello=hello-app:v2`.

Результат ([scenario3-stats.txt](evidence/scenario3-stats.txt)):

```
всего запросов: 991
успешных: 991
ошибок: 0
по версиям:
  v1: 245
  v2: 746
```

Версии по секундам ([scenario3-versions.txt](evidence/scenario3-versions.txt)):

```
23:45:00  version=v1: 20  version=v2: 0    <- set image
23:45:01  version=v1: 20  version=v2: 0
23:45:02  version=v1: 12  version=v2: 7
23:45:03  version=v1: 4  version=v2: 16
23:45:04  version=v1: 0  version=v2: 20
```

Обновление заняло около 2 секунд ([scenario3-rollout.txt](evidence/scenario3-rollout.txt)). Сначала создался один под v2, когда он стал Ready, удалился один v1, потом то же самое со вторым ([scenario3-pods.txt](evidence/scenario3-pods.txt)). Примерно 2 секунды отвечали обе версии сразу. Ошибок нет: из-за `maxUnavailable: 0` рабочих подов всегда было не меньше 2, а старые поды не закрывались сразу из-за preStop.

## Границы гарантий

- Отказ узла. В kind всего один узел, если он упадет, упадет все. Реплики тут не помогут.
- Ошибка в обеих версиях. Если v2 вообще не запустится, обновление застрянет и v1 продолжит работать. Если ошибка есть и в v1, и в v2, откатываться некуда.
- Логическая ошибка. Если `/` отвечает неправильно, а `/live` и `/ready` нормально, Kubernetes ничего не заметит. Это видно только по мониторингу.
- delete pod и eviction. PodDisruptionBudget защищает только от eviction (например `kubectl drain`). От `kubectl delete pod` он не защищает.
- preStop и terminationGracePeriodSeconds. Удаление пода из Endpoints и SIGTERM приложению происходят одновременно. Без preStop приложение может закрыться раньше, чем под уберут из Endpoints, и часть запросов получит ошибку. terminationGracePeriodSeconds по умолчанию 30 секунд, после этого придет SIGKILL. Значит очень долгие запросы все равно оборвутся.
- Not-ready под. Пока проба не сработала (до 4 секунд), запросы на сломанный под еще идут.
