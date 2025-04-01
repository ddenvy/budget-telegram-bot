# Используем официальный образ Go
FROM golang:1.22-alpine

# Устанавливаем необходимые пакеты
RUN apk add --no-cache gcc musl-dev tzdata

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем файлы с зависимостями
COPY go.mod go.sum ./

# Скачиваем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN go build -o bot

# Создаем volume для базы данных
VOLUME ["/app/data"]

# Устанавливаем переменные окружения для прокси
ENV HTTP_PROXY=http://host.docker.internal:26001
ENV HTTPS_PROXY=http://host.docker.internal:26001

# Запускаем бот
CMD ["./bot"] 