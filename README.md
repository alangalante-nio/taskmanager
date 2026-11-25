# taskmanager

Gestor de tarefas pessoal: criar tarefas, definir e alterar a prioridade (Urgente, Alta, Média, Baixa) e marcar como concluída. Backend em Go (só biblioteca padrão), frontend em uma única página HTML embutida no binário, dados em um arquivo JSON.

## Executar com podman

```powershell
podman build -t taskmanager .
```

```powershell
podman run -d --name taskmanager -p 127.0.0.1:8090:8080 -v taskmanager-data:/data taskmanager
```

Abra http://localhost:8090. As tarefas ficam no volume `taskmanager-data` e sobrevivem à recriação do container.

Parar e iniciar de novo:

```powershell
podman stop taskmanager
```

```powershell
podman start taskmanager
```

## Executar sem container

```powershell
go run .
```

Os dados ficam em `data/tasks.json`.

## Configuração

| Variável    | Padrão (container)  | Descrição                    |
|-------------|---------------------|------------------------------|
| `ADDR`      | `:8080`             | Endereço em que o servidor ouve |
| `DATA_FILE` | `/data/tasks.json`  | Arquivo onde as tarefas são salvas |

## API

| Método   | Caminho            | Corpo                                  |
|----------|--------------------|----------------------------------------|
| `GET`    | `/api/tasks`       | —                                      |
| `POST`   | `/api/tasks`       | `{"title": "...", "priority": 1-4}`    |
| `PATCH`  | `/api/tasks/{id}`  | qualquer de `title`, `priority`, `done` |
| `DELETE` | `/api/tasks/{id}`  | —                                      |
