# Shared Budget Bot

A Telegram bot for managing shared expenses and income. Built with Go and SQLite.

## Features

- Track shared expenses and income
- Categorize expenses
- Monthly statistics
- Balance management
- User authorization
- Interactive menu with commands

## Commands

- `/start` - Start the bot and show welcome message
- `/balance` - Show current balance
- `/add_income <amount>` - Add income
- `/set_balance <amount>` - Set new balance
- `/summary` - Show monthly statistics
- `/reset_balance` - Reset balance to zero

## Expense Categories

- Food
- House
- Transportation
- Grocery
- Entertainment
- MonicaBB
- Emergency

## Setup

1. Create a `.env` file with your Telegram bot token:
```
BOT_TOKEN=your_bot_token_here
```

2. Build and run with Docker:
```bash
docker-compose up --build -d
```

## Authorized Users

Only the following users have access to the bot:
- @envydany
- @TANIAPENG

## Adding Expenses

1. Send the amount (e.g., "100")
2. Select a category from the provided list
3. Enter a description of the purchase

The bot will then show:
- Category
- Description
- Amount
- Date
- User
- Total balance
- Monthly expenses

## Database

The bot uses SQLite for data storage. The database file is located at `data/budget.db`.

## Development

The bot is written in Go and uses the following main packages:
- github.com/go-telegram-bot-api/telegram-bot-api/v5
- github.com/joho/godotenv 