package main

import (
	"fmt"
	"html"
	"log"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/kyokomi/emoji"
)

func postPoll(bot *tgbotapi.BotAPI, p *poll, chatid int64) (tgbotapi.Message, error) {
	share := tgbotapi.InlineKeyboardButton{
		Text:              locSharePoll,
		SwitchInlineQuery: &p.Question,
	}
	new := tgbotapi.NewInlineKeyboardButtonData(locCreateNewPoll, createPollQuery)

	buttons := tgbotapi.NewInlineKeyboardRow(share, new)
	markup := tgbotapi.NewInlineKeyboardMarkup(buttons)
	messageTxt := locFinishedCreatingPoll
	messageTxt += p.Question + "\n\n"

	for i, o := range p.Options {
		messageTxt += strconv.Itoa(i+1) + ") " + o.Text + "\n"
	}
	msg := tgbotapi.NewMessage(chatid, messageTxt)
	msg.ReplyMarkup = markup

	return bot.Send(msg)
}

func sendMainMenuMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update) (tgbotapi.Message, error) {
	buttons := make([]tgbotapi.InlineKeyboardButton, 0)
	buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("create poll", createPollQuery))
	markup := tgbotapi.NewInlineKeyboardMarkup(buttons)
	messageTxt := locMainMenu
	msg := tgbotapi.NewMessage(int64(update.Message.From.ID), messageTxt)
	msg.ReplyMarkup = markup

	return bot.Send(msg)
}

func sendListChatsMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update, st Store) error {
	chats, err := st.GetUserChatIds(update.Message.From.ID)
	_, _, chatID, _, err := st.GetState(update.Message.From.ID)
	if err != nil {
		return err
	}
	buttonrows := make([][]tgbotapi.InlineKeyboardButton, 0)

	for _, chat := range chats {
		buttonLabel := fmt.Sprintf("%s", chat.Title)
		if chat.ID == chatID {
			buttonLabel = "-> " + buttonLabel
		}
		buttons := make([]tgbotapi.InlineKeyboardButton, 0)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(buttonLabel, fmt.Sprintf("chat:%d", chat.ID)))
		buttonrows = append(buttonrows, buttons)
	}
	markup := tgbotapi.NewInlineKeyboardMarkup(buttonrows...)
	messageTxt := locChatsListMessage
	msg := tgbotapi.NewMessage(int64(update.Message.From.ID), messageTxt)
	msg.ReplyMarkup = markup
	bot.Send(msg)
	return nil
}

func sendListUsersMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update, st Store) error {
	_, _, chatID, userContext, err := st.GetState(update.CallbackQuery.From.ID)
	users, err := st.GetChatUsers(chatID)
	if err != nil {
		return err
	}
	buttonrows := make([][]tgbotapi.InlineKeyboardButton, 0)
	for _, user := range users {
		u := tgbotapi.User{
			ID:        user.ID,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			UserName:  user.UserName,
		}
		buttonLabel := fmt.Sprintf("%s (%s) P:%d T:%s", html.EscapeString(getDisplayUserName(&u)), html.EscapeString(user.NameOverride), user.Priority, user.Tag)
		if userContext == user.ID {
			buttonLabel = "-> " + buttonLabel
		}
		buttons := make([]tgbotapi.InlineKeyboardButton, 0)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(
			buttonLabel,
			fmt.Sprintf("player:%d:%d", user.ID, chatID)))
		buttonrows = append(buttonrows, buttons)
	}
	markup := tgbotapi.NewInlineKeyboardMarkup(buttonrows...)
	messageTxt := locChatsListMessage
	msg := tgbotapi.NewMessage(int64(update.CallbackQuery.From.ID), messageTxt)
	msg.ReplyMarkup = markup
	bot.Send(msg)
	return nil
}

func sendInterMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update, p *poll) (tgbotapi.Message, error) {
	//shareButton := tgbotapi.InlineKeyboardButton{
	//Text:              locSharePoll,
	//SwitchInlineQuery: &p.Question,
	//}
	pollDoneButton := tgbotapi.NewInlineKeyboardButtonData(
		locPollDoneButton, fmt.Sprintf("%s:%d", pollDoneQuery, p.ID))

	buttons := make([]tgbotapi.InlineKeyboardButton, 0)
	buttons = append(buttons, pollDoneButton)
	//buttons = append(buttons, shareButton)

	markup := tgbotapi.NewInlineKeyboardMarkup(buttons)
	messageTxt := locAddedOption
	messageTxt += p.Question + "\n\n"

	for i, o := range p.Options {
		messageTxt += strconv.Itoa(i+1) + ") " + o.Text + "\n"
	}
	msg := tgbotapi.NewMessage(int64(update.Message.From.ID), messageTxt)
	msg.ReplyMarkup = markup

	return bot.Send(msg)
}

func sendNewQuestionMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update, st Store) error {
	msg := tgbotapi.NewMessage(int64(update.CallbackQuery.From.ID), locNewQuestion)
	_, err := bot.Send(&msg)
	if err != nil {
		return fmt.Errorf("could not send message: %v", err)
	}
	state, pollid, chatID, userContext, err := st.GetState(update.CallbackQuery.From.ID)
	if err != nil {
		chatID = -1
	}
	state = waitingForQuestion
	err = st.SaveState(update.CallbackQuery.From.ID, pollid, state, chatID, userContext)
	if err != nil {
		return fmt.Errorf("could not change state to waiting for questions: %v", err)
	}
	return nil
}

func sendEditMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update, p *poll) (tgbotapi.Message, error) {
	body := "This is the poll currently selected:\n<pre>\n"
	body += p.Question + "\n"
	for i, o := range p.Options {
		body += fmt.Sprintf("%d. %s", i+1, o.Text) + "\n"
	}
	body += "</pre>\n\n"
	msg := tgbotapi.NewMessage(int64(update.Message.From.ID), body)
	msg.ParseMode = tgbotapi.ModeHTML

	msg.ReplyMarkup = buildEditMarkup(p, false, false)

	return bot.Send(&msg)
}

func buildPollMarkup(p *poll) *tgbotapi.InlineKeyboardMarkup {
	buttonrows := make([][]tgbotapi.InlineKeyboardButton, 0) //len(p.Options), len(p.Options))
	row := -1

	votesForOption := make(map[int]int)
	for _, o := range p.Options {
		for _, a := range p.Answers {
			if a.OptionID == o.ID {
				votesForOption[o.ID]++
			}
		}
	}
	cnt := 0
	for _, o := range p.Options {
		textWidth := 0
		if row != -1 {
			for _, b := range buttonrows[row] {
				textWidth += len(b.Text)
			}
		}
		textWidth += len(o.Text)
		if cnt == 0 || cnt == 3 {
			row++
			buttonrows = append(buttonrows, make([]tgbotapi.InlineKeyboardButton, 0))
		}
		label := fmt.Sprintf("%s (%d)", o.Text, votesForOption[o.ID])
		callback := fmt.Sprintf("%d:%d", p.ID, o.ID)
		button := tgbotapi.NewInlineKeyboardButtonData(label, callback)
		cnt++
		buttonrows[row] = append(buttonrows[row], button)
	}
	markup := tgbotapi.NewInlineKeyboardMarkup(buttonrows...)

	return &markup
}

func buildPollListing(p *poll, st Store) (listing string) {
	// polledUsers := make(map[int]struct{})
	listOfUsers := make([][]user, len(p.Options))
	var backupPlayers []user
	fieldPlayerCount := 0
	votesForOption := make(map[int]int)
	for _, a := range p.Answers {
		for i, o := range p.Options {
			if a.OptionID == o.ID {
				votesForOption[o.ID]++
				u, err := st.GetPlayer(a.UserID, p.ChatID)
				if i <= 2 {
					fieldPlayerCount++
				}
				if err != nil {
					log.Printf("could not get user: %v", err)
					if i > 2 || fieldPlayerCount <= maxPlayersInTeams {
						listOfUsers[i] = append(listOfUsers[i], user{ID: a.UserID})
					} else {
						backupPlayers = append(backupPlayers, user{ID: a.UserID})
					}
					continue
				}
				if i > 2 || fieldPlayerCount <= maxPlayersInTeams {
					// polledUsers[u.ID] = struct{}{}
					listOfUsers[i] = append(listOfUsers[i], u)
				} else {
					backupPlayers = append(backupPlayers, u)
				}
			}
		}
	}

	if p.Type == typeGame {
		listing += emoji.Sprintf(":volleyball:<b>%s</b>\n", html.EscapeString(p.Question))
		//log.Printf("Create listing for question: %s\n", p.Question)
		opts := [3]string{emoji.Sprintf(":red_circle:"), emoji.Sprintf(":white_circle:"), emoji.Sprintf(":blue_circle:")}
		mainPlayerCount := 0
		for i := 0; i <= 2; i++ {
			var part string
			if len(p.Answers) > 0 {
				part = fmt.Sprintf("%d -", len(listOfUsers[i])) //fmt.Sprintf(" (%.0f%%)", 100.*float64(votesForOption[o.ID])/float64(len(polledUsers)))
				if votesForOption[p.Options[i].ID] != p.Options[i].Ctr {
					log.Printf("Counter for option #%d is off: %d stored vs. %d counted", p.Options[i].ID, p.Options[i].Ctr, votesForOption[p.Options[i].ID])
				}
			}
			if len(listOfUsers[i]) > 0 {
				listing += fmt.Sprintf("\n<b>%s %s </b>", html.EscapeString(opts[i]), part)
			}

			usersOnAnswer := len(listOfUsers[i])
			if len(p.Answers) < maxNumberOfUsersListed && usersOnAnswer > 0 {

				listing += "" + html.EscapeString(getDisplayUserName2(listOfUsers[i][0]))
				mainPlayerCount++
				for j := 1; j+1 <= usersOnAnswer; j++ {
					listing += ", " + html.EscapeString(getDisplayUserName2(listOfUsers[i][j]))
					mainPlayerCount++

				}
			}
			if len(listOfUsers[i]) > 0 {
				listing += "\n"
			}
		}

		if len(backupPlayers) > 0 {
			listing += "\n<b>В запасе:</b> "
			for i, user := range backupPlayers {
				listing += html.EscapeString(getDisplayUserName2(user))
				if i+1 != len(backupPlayers) {
					listing += ", "
				} else {
					listing += "\n"
				}
			}
		}

		var addPlayers = make(map[string]int)
		addPlayerCount := 0
		for _, u := range listOfUsers[3] {
			addPlayers[getDisplayUserName2(u)] = addPlayers[getDisplayUserName2(u)] + 1
			addPlayerCount++
		}
		for _, u := range listOfUsers[4] {
			addPlayers[getDisplayUserName2(u)] = addPlayers[getDisplayUserName2(u)] + 2
			addPlayerCount += 2
		}
		if addPlayerCount > 0 {
			listing += "\n<b>Приглашенные:</b> "
			for user, added := range addPlayers {
				listing += html.EscapeString(user) + "(" + fmt.Sprintf("%d", added) + ") "
			}
		}

		totalPlayers := mainPlayerCount + addPlayerCount + len(backupPlayers)
		listing += emoji.Sprint(fmt.Sprintf("\n%d :busts_in_silhouette:\n", totalPlayers))
	}
	return listing
}

func buildEditMarkup(p *poll, noOlder, noNewer bool) *tgbotapi.InlineKeyboardMarkup {
	query := fmt.Sprintf("e:%d", p.ID)

	buttonrows := make([][]tgbotapi.InlineKeyboardButton, 0)
	buttonrows = append(buttonrows, make([]tgbotapi.InlineKeyboardButton, 0))
	buttonrows = append(buttonrows, make([]tgbotapi.InlineKeyboardButton, 0))
	buttonrows = append(buttonrows, make([]tgbotapi.InlineKeyboardButton, 0))

	buttonLast := tgbotapi.NewInlineKeyboardButtonData("\u2B05", query+":-")
	buttonNext := tgbotapi.NewInlineKeyboardButtonData("\u27A1", query+":+")
	if noOlder {
		buttonLast = tgbotapi.NewInlineKeyboardButtonData("\u2B05", "dummy")
	}
	if noNewer {
		buttonNext = tgbotapi.NewInlineKeyboardButtonData("\u27A1", "dummy")
	}
	buttonrows[0] = append(buttonrows[0], buttonLast, buttonNext)
	buttonInactive := tgbotapi.NewInlineKeyboardButtonData(locToggleOpen, query+":c")
	if isInactive(p) {
		buttonInactive = tgbotapi.NewInlineKeyboardButtonData(locToggleInactive, query+":c")
	}
	buttonMultipleChoice := tgbotapi.NewInlineKeyboardButtonData(locToggleSingleChoice, query+":m")
	// if isMultipleChoice(p) {
	// 	buttonMultipleChoice = tgbotapi.NewInlineKeyboardButtonData(locToggleMultipleChoice, query+":m")
	// }
	buttonrows[1] = append(buttonrows[1], buttonInactive)
	if !isMultipleChoice(p) {
		buttonrows[1] = append(buttonrows[1], buttonMultipleChoice)
	}
	buttonEditQuestion := tgbotapi.NewInlineKeyboardButtonData(locEditQuestionButton, query+":q")
	buttonAddOptions := tgbotapi.NewInlineKeyboardButtonData(locAddOptionButton, query+":o")

	buttonrows[2] = append(buttonrows[2], buttonEditQuestion, buttonAddOptions)
	markup := tgbotapi.NewInlineKeyboardMarkup(buttonrows...)

	return &markup
}

func sendPlayerMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update, user user, fromID int) (tgbotapi.Message, error) {
	u := tgbotapi.User{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		UserName:  user.UserName,
	}

	body := "Player settings:\n<pre>\n"
	body += html.EscapeString(getDisplayUserName(&u)) + "\n"
	body += "Overriden name: " + html.EscapeString(user.NameOverride) + "\n"
	body += "Vote priority: " + fmt.Sprintf("%d", user.Priority) + "\n"
	body += "Tag: " + user.Tag + "\n"
	body += "</pre>\n\n"
	msg := tgbotapi.NewMessage(int64(fromID), body)
	msg.ParseMode = tgbotapi.ModeHTML

	msg.ReplyMarkup = buildPlayerMarkup(user)

	return bot.Send(&msg)
}

func buildPlayerMarkup(user user) *tgbotapi.InlineKeyboardMarkup {
	query := fmt.Sprintf("%d:%d", user.ID, user.ChatID)

	buttonRows := make([][]tgbotapi.InlineKeyboardButton, 0)
	buttonRows = append(buttonRows, make([]tgbotapi.InlineKeyboardButton, 0))
	buttonRows[0] = append(buttonRows[0], tgbotapi.NewInlineKeyboardButtonData("Tag", "playertag:"+query))
	buttonRows[0] = append(buttonRows[0], tgbotapi.NewInlineKeyboardButtonData("Priority", "playerpriority:"+query))
	buttonRows[0] = append(buttonRows[0], tgbotapi.NewInlineKeyboardButtonData("Name", "playername:"+query))
	markup := tgbotapi.NewInlineKeyboardMarkup(buttonRows...)
	return &markup
}

func getDisplayUserName(u *tgbotapi.User) string {
	if u.FirstName == "" && u.LastName == "" {
		return strconv.Itoa(u.ID)
	} else if u.FirstName != "" {
		name := u.FirstName
		if u.LastName != "" {
			name += " " + u.LastName
		}
		return name
	}
	return u.LastName
}

func getDisplayUserName2(u user) string {
	usr := tgbotapi.User{
		ID:        u.ID,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		UserName:  u.UserName,
	}
	ret := getDisplayUserName(&usr)
	if u.NameOverride != "" {
		ret = u.NameOverride
	}
	if u.Tag != "" {
		ret = ret + " " + u.Tag
	}
	return ret
}
