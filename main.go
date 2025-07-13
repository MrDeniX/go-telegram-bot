package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "strings"
    "github.com/joho/godotenv"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type UserSession struct {
    Step string
    Data map[string]string
}

var userState = make(map[int64]string)
var sessions = make(map[int64]*UserSession)

func main() {
    
    err := godotenv.Load()
    if err != nil {
        log.Println("Warning: Не удалось загрузить .env файл")
    }

    bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
    if err != nil {
        log.Panic(err)
    }

    bot.Debug = true
    log.Printf("Бот запущен: @%s", bot.Self.UserName)

    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60
    updates := bot.GetUpdatesChan(u)

    for update := range updates {
        if update.Message == nil {
            continue
        }

        chatID := update.Message.Chat.ID
        text := update.Message.Text
        username := update.Message.From.UserName

        session, hasSession := sessions[chatID]

        if hasSession {
            switch session.Step {
            case "waiting_course_subject":
                session.Data["Предмет"] = text
                session.Step = "waiting_course_site"
                bot.Send(tgbotapi.NewMessage(chatID, "Есть ли у вас готовый сайт? (Да / Нет)"))
                continue
            case "waiting_course_site":
                session.Data["Готовый сайт"] = text
                saveToFile(username, chatID, "Курсовая", session.Data)
                delete(sessions, chatID)
                bot.Send(tgbotapi.NewMessage(chatID, "Спасибо! Данные по курсовой записаны."))
                continue
            case "waiting_practice_subject":
                session.Data["Предмет"] = text
                session.Step = "waiting_practice_site"
                bot.Send(tgbotapi.NewMessage(chatID, "Есть ли у вас готовый сайт? (Да / Нет)"))
                continue
            case "waiting_practice_site":
                session.Data["Готовый сайт"] = text
                saveToFile(username, chatID, "Практика", session.Data)
                delete(sessions, chatID)
                bot.Send(tgbotapi.NewMessage(chatID, "Отлично! Практика записана."))
                continue
            case "waiting_teacher_name":
                session.Data["Преподаватель"] = text
                session.Step = "waiting_course_number"
                bot.Send(tgbotapi.NewMessage(chatID, "Укажите курс (например, 2 курс):"))
                continue
            case "waiting_course_number":
                session.Data["Курс"] = text
                saveToFile(username, chatID, "Практические задания", session.Data)
                delete(sessions, chatID)
                bot.Send(tgbotapi.NewMessage(chatID, "Спасибо! Данные по заданиям сохранены."))
                continue
            }
        }

        switch text {
        case "/start":
            sendMainMenu(bot, chatID)
        case "📚 Курсовая":
            sessions[chatID] = &UserSession{Step: "waiting_course_subject", Data: make(map[string]string)}
            bot.Send(tgbotapi.NewMessage(chatID, "Введите предмет для курсовой:"))
        case "🧪 Практика":
            sessions[chatID] = &UserSession{Step: "waiting_practice_subject", Data: make(map[string]string)}
            bot.Send(tgbotapi.NewMessage(chatID, "Введите предмет практики:"))
        case "📝 Практические задания":
            sessions[chatID] = &UserSession{Step: "waiting_teacher_name", Data: make(map[string]string)}
            bot.Send(tgbotapi.NewMessage(chatID, "Введите имя преподавателя:"))
        case "☁️ Погода":
            weather := getWeather("Simferopol")
            bot.Send(tgbotapi.NewMessage(chatID, weather))
        default:
            bot.Send(tgbotapi.NewMessage(chatID, "Пожалуйста, выбери пункт из меню или введи /start."))
        }
    }
}

func sendMainMenu(bot *tgbotapi.BotAPI, chatID int64) {
    keyboard := tgbotapi.NewReplyKeyboard(
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButton("📚 Курсовая"),
            tgbotapi.NewKeyboardButton("📝 Практические задания"),
        ),
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButton("🧪 Практика"),
            tgbotapi.NewKeyboardButton("☁️ Погода"),
        ),
    )

    msg := tgbotapi.NewMessage(chatID, "Выбери один из пунктов меню:")
    msg.ReplyMarkup = keyboard
    bot.Send(msg)
}

func getWeather(city string) string {
    apiKey := os.Getenv("OPENWEATHER_TOKEN")
    if apiKey == "" {
        return "API ключ OpenWeather не задан."
    }

    url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=metric&lang=ru", city, apiKey)
    resp, err := http.Get(url)
    if err != nil {
        return "Ошибка при получении погоды."
    }
    defer resp.Body.Close()

    var data map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&data)

    if data["main"] == nil {
        return "Не удалось получить данные о погоде."
    }

    main := data["main"].(map[string]interface{})
    weatherArr := data["weather"].([]interface{})
    description := weatherArr[0].(map[string]interface{})["description"].(string)
    temp := main["temp"].(float64)

    return fmt.Sprintf("🌤 Погода в Симферополе:\n%s, %.1f°C", strings.Title(description), temp)
}

func saveToFile(username string, chatID int64, title string, fields map[string]string) {
    file, err := os.OpenFile("data.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Println("Ошибка записи в файл:", err)
        return
    }
    defer file.Close()

    if username == "" {
        username = "(без username)"
    }

    fmt.Fprintf(file, "[Username: %s | UserID: %d]\nТип: %s\n", username, chatID, title)
    for k, v := range fields {
        fmt.Fprintf(file, "%s: %s\n", k, v)
    }
    fmt.Fprintln(file, "---")
}

