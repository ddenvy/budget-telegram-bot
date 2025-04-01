package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"telegram-budget-bot/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

var (
	db            *database.DB
	expenseRegexp = regexp.MustCompile(`^(.*?)\s+(\d+(?:\.\d{1,2})?)\s*(?:CNY)?$`)
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	var err error
	db, err = database.NewDB("data/budget.db")
	if err != nil {
		log.Fatal(err)
	}
}

func handleStart(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	err := db.RegisterUser(message.From.ID, message.From.UserName)
	if err != nil {
		log.Printf("Error registering user: %v", err)
		return
	}

	balance, err := db.GetBalance()
	if err != nil {
		log.Printf("Error getting balance: %v", err)
		return
	}

	response := fmt.Sprintf(
		"Welcome to Shared Budget Bot! 🎉\n"+
			"Current balance: %.2f CNY\n\n"+
			"Available commands:\n"+
			"/balance - Show current balance\n"+
			"/add_income <amount> - Add income\n"+
			"/set_balance <amount> - Set new balance\n"+
			"/summary - Show monthly summary\n\n"+
			"To add expense, simply send: description amount\n"+
			"Example: bread 10",
		balance,
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleBalance(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	balance, err := db.GetBalance()
	if err != nil {
		log.Printf("Error getting balance: %v", err)
		return
	}

	response := fmt.Sprintf("Current shared balance: %.2f CNY", balance)
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleAddIncome(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := strings.Fields(message.Text)
	if len(args) != 2 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Please use format: /add_income <amount>\nExample: /add_income 1000")
		bot.Send(msg)
		return
	}

	amount, err := strconv.ParseFloat(args[1], 64)
	if err != nil || amount <= 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Please provide a valid positive number")
		bot.Send(msg)
		return
	}

	err = db.AddTransaction(message.From.ID, "income", amount, "Income")
	if err != nil {
		log.Printf("Error adding income: %v", err)
		return
	}

	balance, err := db.GetBalance()
	if err != nil {
		log.Printf("Error getting balance: %v", err)
		return
	}

	response := fmt.Sprintf(
		"Income added: +%.2f CNY\n"+
			"Shared balance: %.2f CNY",
		amount, balance,
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleSetBalance(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := strings.Fields(message.Text)
	if len(args) != 2 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Please use format: /set_balance <amount>\nExample: /set_balance 1000")
		bot.Send(msg)
		return
	}

	newBalance, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Please provide a valid number")
		bot.Send(msg)
		return
	}

	currentBalance, err := db.GetBalance()
	if err != nil {
		log.Printf("Error getting balance: %v", err)
		return
	}

	difference := newBalance - currentBalance
	if difference != 0 {
		transType := "income"
		amount := difference
		if difference < 0 {
			transType = "expense"
			amount = -difference
		}

		err = db.AddTransaction(message.From.ID, transType, amount, "Balance adjustment")
		if err != nil {
			log.Printf("Error adjusting balance: %v", err)
			return
		}
	}

	response := fmt.Sprintf("Balance has been set to: %.2f CNY", newBalance)
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleSummary(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	income, expenses, balance, err := db.GetMonthlyStats()
	if err != nil {
		log.Printf("Error getting monthly stats: %v", err)
		return
	}

	response := fmt.Sprintf(
		"📊 Monthly Summary\n\n"+
			"Income: +%.2f CNY\n"+
			"Expenses: -%.2f CNY\n"+
			"Current Balance: %.2f CNY",
		income, expenses, balance,
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleExpense(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	matches := expenseRegexp.FindStringSubmatch(message.Text)
	if matches == nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"Invalid format. Please use: description amount\nExample: bread 10",
		)
		bot.Send(msg)
		return
	}

	description := strings.TrimSpace(matches[1])
	amount, err := strconv.ParseFloat(matches[2], 64)
	if err != nil || amount <= 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Please provide a valid positive number")
		bot.Send(msg)
		return
	}

	err = db.AddTransaction(message.From.ID, "expense", amount, description)
	if err != nil {
		log.Printf("Error adding expense: %v", err)
		return
	}

	balance, err := db.GetBalance()
	if err != nil {
		log.Printf("Error getting balance: %v", err)
		return
	}

	monthlyExpenses, err := db.GetMonthlyExpenses()
	if err != nil {
		log.Printf("Error getting monthly expenses: %v", err)
		return
	}

	response := fmt.Sprintf(
		"Expense added: %s - %.2f CNY\n"+
			"Shared balance: %.2f CNY\n"+
			"Spent this month: %.2f CNY",
		description, amount, balance, monthlyExpenses,
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN must be set")
	}

	// Создаем бота
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		switch {
		case update.Message.Command() == "start":
			handleStart(bot, update.Message)
		case update.Message.Command() == "balance":
			handleBalance(bot, update.Message)
		case update.Message.Command() == "add_income":
			handleAddIncome(bot, update.Message)
		case update.Message.Command() == "set_balance":
			handleSetBalance(bot, update.Message)
		case update.Message.Command() == "summary":
			handleSummary(bot, update.Message)
		case update.Message.Text != "":
			handleExpense(bot, update.Message)
		}
	}
}
