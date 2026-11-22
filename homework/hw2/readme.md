# ДЗ 2. Управление изменениями

## Приложение

В `app/` лежит версия приложения из ДЗ 1 с флагом `new-greeting`. После запуска флаг выключен. `GET /message` возвращает старое приветствие, а после `GET /admin/feature?enabled=true` возвращает новое. `GET /admin/feature?enabled=false` возвращает старое поведение. `GET /feature` показывает текущее состояние. Некорректное значение `enabled` дает HTTP 400 и оставляет состояние прежним.

## CI

Workflow `.github/workflows/ci.yml` выполняет `go test -v ./...` и `docker build` на push и pull request. Тесты проверяют начальное состояние, включение, выключение и некорректный ввод. Тег образа содержит SHA коммита, сборка остается на runner.

Из корня репозитория можно повторить те же шаги локально.

```bash
sha=$(git rev-parse HEAD)
(cd homework/hw2/app && go test -v ./...)
docker build --build-arg VERSION="$sha" -t "hello-hw2:$sha" homework/hw2/app
```

В GitHub Actions для коммита `683f908` оба теста прошли и Docker-образ собрался. Локальный запуск тестов и сборка также прошли успешно. [Запуск GitHub Actions](https://github.com/KozlovNA/sre-course/actions/runs/44880200706)

## Проверка в kind

Из корня репозитория, если кластер kind запущен.

```bash
docker build --build-arg VERSION=hw2 -t hello-hw2:local homework/hw2/app
kind load docker-image hello-hw2:local --name kind
k=(kubectl --context kind-kind --namespace default)
"${k[@]}" apply -f homework/hw2/kubernetes/deployment.yaml
"${k[@]}" rollout status deployment/hello-hw2
pod=$("${k[@]}" get pods -l app=hello-hw2 -o jsonpath='{.items[0].metadata.name}')
"${k[@]}" port-forward --address 127.0.0.1 "pod/$pod" 18080:8080
```

В другом терминале выполнить запросы и сравнить имя, UID и число перезапусков Pod.

```bash
k=(kubectl --context kind-kind --namespace default)
pod=$("${k[@]}" get pods -l app=hello-hw2 -o jsonpath='{.items[0].metadata.name}')
pod_state() {
    "${k[@]}" get "pod/$pod" -o jsonpath='{.metadata.name}{" uid="}{.metadata.uid}{" restarts="}{.status.containerStatuses[0].restartCount}{"\n"}'
}
url=http://127.0.0.1:18080
pod_state
curl -fsS "$url/feature"
curl -fsS "$url/message"
curl -fsS "$url/admin/feature?enabled=true"
curl -fsS "$url/feature"
curl -fsS "$url/message"
pod_state
curl -fsS "$url/admin/feature?enabled=false"
curl -fsS "$url/feature"
curl -fsS "$url/message"
pod_state
```

Результат проверки в Kubernetes.

```text
Pod before: hello-hw2-7dff7c95d5-pwqrn uid=37435720-7470-4324-a603-69053ba5a905 restarts=0
GET /feature -> new-greeting=false
GET /message -> hello version=hw2 pod=hello-hw2-7dff7c95d5-pwqrn
GET /admin/feature?enabled=true -> new-greeting=true
GET /feature -> new-greeting=true
GET /message -> hello from the new greeting version=hw2 pod=hello-hw2-7dff7c95d5-pwqrn
Pod enabled: hello-hw2-7dff7c95d5-pwqrn uid=37435720-7470-4324-a603-69053ba5a905 restarts=0
GET /admin/feature?enabled=false -> new-greeting=false
GET /feature -> new-greeting=false
GET /message -> hello version=hw2 pod=hello-hw2-7dff7c95d5-pwqrn
Pod after: hello-hw2-7dff7c95d5-pwqrn uid=37435720-7470-4324-a603-69053ba5a905 restarts=0
```

## Вывод

Выкладка нового образа доставляет код в Kubernetes. Флаг меняет поведение уже запущенного процесса, поэтому сборка образа и новая выкладка для переключения не нужны. Имя и UID Pod совпали при всех проверках, число перезапусков осталось 0.

Флаг хранится в памяти процесса и сбрасывается после его перезапуска. У каждой реплики было бы свое состояние, поэтому в учебной выкладке одна реплика. Доступ для проверки идет через `port-forward` на `127.0.0.1`. У Deployment нет Service или Ingress.
