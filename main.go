package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"telegram-budget-bot/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

var (
	db            *database.DB
	expenseRegexp = regexp.MustCompile(`^(\d+(?:\.\d{1,2})?)$`)
	allowedUsers  = map[string]bool{
		"envydany":  true,
		"TANIAPENG": true,
	}
	userStates = make(map[int64]string)
	categories = []string{
		"Food", "House", "Transportation", "Grocery",
		"Entertainment", "MonicaBB", "Emergency",
	}
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

func isAuthorized(username string) bool {
	return allowedUsers[username]
}

func handleStart(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if !isAuthorized(message.From.UserName) {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Sorry, you don't have access to this bot.")
		bot.Send(msg)
		return
	}

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
			"/summary - Show monthly statistics\n"+
			"/reset_balance - Reset balance to zero\n\n"+
			"To add an expense, just send the amount.",
		balance,
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleResetBalance(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if !isAuthorized(message.From.UserName) {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Sorry, you don't have access to this command.")
		bot.Send(msg)
		return
	}

	err := db.ResetBalance()
	if err != nil {
		log.Printf("Error resetting balance: %v", err)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "Balance has been successfully reset to zero!")
	bot.Send(msg)
}

func handleBalance(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if !isAuthorized(message.From.UserName) {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Sorry, you don't have access to this command.")
		bot.Send(msg)
		return
	}

	balance, err := db.GetBalance()
	if err != nil {
		log.Printf("Error getting balance: %v", err)
		return
	}

	response := fmt.Sprintf("Current total balance: %.2f CNY", balance)
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleAddIncome(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if !isAuthorized(message.From.UserName) {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Sorry, you don't have access to this command.")
		bot.Send(msg)
		return
	}

	args := strings.Fields(message.Text)
	if len(args) != 2 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Please use the format: /add_income <amount>\nExample: /add_income 1000")
		bot.Send(msg)
		return
	}

	amount, err := strconv.ParseFloat(args[1], 64)
	if err != nil || amount <= 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Please enter a valid positive number")
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
			"Total balance: %.2f CNY",
		amount, balance,
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleSetBalance(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if !isAuthorized(message.From.UserName) {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Sorry, you don't have access to this command.")
		bot.Send(msg)
		return
	}

	args := strings.Fields(message.Text)
	if len(args) != 2 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Please use the format: /set_balance <amount>\nExample: /set_balance 1000")
		bot.Send(msg)
		return
	}

	newBalance, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Please enter a valid number")
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
	if !isAuthorized(message.From.UserName) {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Sorry, you don't have access to this command.")
		bot.Send(msg)
		return
	}

	income, expenses, balance, err := db.GetMonthlyStats()
	if err != nil {
		log.Printf("Error getting monthly stats: %v", err)
		return
	}

	response := fmt.Sprintf(
		"📊 Monthly Statistics\n\n"+
			"Income: +%.2f CNY\n"+
			"Expenses: -%.2f CNY\n"+
			"Current balance: %.2f CNY",
		income, expenses, balance,
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func showCategories(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for _, category := range categories {
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(category, fmt.Sprintf("category:%s", category)),
		})
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "Select a category:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	bot.Send(msg)
}

func handleExpense(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if !isAuthorized(message.From.UserName) {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Sorry, you don't have access to this command.")
		bot.Send(msg)
		return
	}

	matches := expenseRegexp.FindStringSubmatch(message.Text)
	if matches == nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"Invalid format. Please send only the amount.\nExample: 100",
		)
		bot.Send(msg)
		return
	}

	amount, err := strconv.ParseFloat(matches[1], 64)
	if err != nil || amount <= 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Please enter a valid positive number")
		bot.Send(msg)
		return
	}

	userStates[message.From.ID] = fmt.Sprintf("amount:%f", amount)
	showCategories(bot, message)
}

func handleCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	if !isAuthorized(callback.From.UserName) {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "Sorry, you don't have access to this command.")
		bot.Send(msg)
		return
	}

	callbackData := callback.Data
	if strings.HasPrefix(callbackData, "category:") {
		category := strings.TrimPrefix(callbackData, "category:")
		_, exists := userStates[callback.From.ID]
		if !exists {
			msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "Error: amount not found. Please start over.")
			bot.Send(msg)
			return
		}

		userStates[callback.From.ID] = fmt.Sprintf("category:%s", category)
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "Now enter the purchase description:")
		bot.Send(msg)
	}
}

func handleDescription(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if !isAuthorized(message.From.UserName) {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Sorry, you don't have access to this command.")
		bot.Send(msg)
		return
	}

	state, exists := userStates[message.From.ID]
	if !exists {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Error: state not found. Please start over.")
		bot.Send(msg)
		return
	}

	if !strings.HasPrefix(state, "category:") {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Error: category not selected. Please start over.")
		bot.Send(msg)
		return
	}

	category := strings.TrimPrefix(state, "category:")
	amountStr := strings.TrimPrefix(userStates[message.From.ID], "amount:")
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Error: invalid amount format.")
		bot.Send(msg)
		return
	}

	description := fmt.Sprintf("%s - %s", category, message.Text)
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
		"Expense added:\n"+
			"Category: %s\n"+
			"Description: %s\n"+
			"Amount: %.2f CNY\n"+
			"Date: %s\n"+
			"User: %s\n\n"+
			"Total balance: %.2f CNY\n"+
			"Monthly expenses: %.2f CNY",
		category, message.Text, amount,
		time.Now().Format("02.01.2006 15:04"),
		message.From.UserName,
		balance, monthlyExpenses,
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)

	// Clear user state
	delete(userStates, message.From.ID)
}

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN must be set")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	// Set up commands menu
	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "Start the bot"},
		{Command: "balance", Description: "Show current balance"},
		{Command: "add_income", Description: "Add income"},
		{Command: "set_balance", Description: "Set new balance"},
		{Command: "summary", Description: "Show monthly statistics"},
		{Command: "reset_balance", Description: "Reset balance to zero"},
	}

	_, err = bot.Request(tgbotapi.NewSetMyCommands(commands...))
	if err != nil {
		log.Printf("Error setting commands: %v", err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil && update.CallbackQuery == nil {
			continue
		}

		if update.CallbackQuery != nil {
			handleCallback(bot, update.CallbackQuery)
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		// Handle commands first
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				handleStart(bot, update.Message)
			case "balance":
				handleBalance(bot, update.Message)
			case "add_income":
				handleAddIncome(bot, update.Message)
			case "set_balance":
				handleSetBalance(bot, update.Message)
			case "summary":
				handleSummary(bot, update.Message)
			case "reset_balance":
				handleResetBalance(bot, update.Message)
			}
			continue
		}

		// Handle user state
		if state, exists := userStates[update.Message.From.ID]; exists {
			if strings.HasPrefix(state, "category:") {
				handleDescription(bot, update.Message)
			}
			continue
		}

		// Handle expense input
		if update.Message.Text != "" {
			handleExpense(bot, update.Message)
		}
	}
}
