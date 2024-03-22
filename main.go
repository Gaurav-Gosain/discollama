package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	"github.com/gaurav-gosain/discollama/utils/ollama"
)

// Bot parameters
var (
	GuildID        = flag.String("guild", "", "Test guild ID. If not passed - bot registers commands globally")
	BotToken       = flag.String("token", "", "Bot access token")
	RemoveCommands = flag.Bool("rmcmd", true, "Remove all commands after shutdowning or not")
	OllamaURL      = flag.String("ollama-url", "http://localhost:11434", "Ollama API URL")
)

var s *discordgo.Session
var api *ollama.APIClient

func init() {
	flag.Parse()
	api = ollama.NewAPIClient(*OllamaURL)
}

func init() {
	var err error
	s, err = discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
}

func init() {
	commandHandlers := map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"generate": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Access options in the order provided by the user.
			options := i.ApplicationCommandData().Options

			// Or convert the slice into a map
			optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
			for _, opt := range options {
				optionMap[opt.Name] = opt
			}

			command := optionMap["prompt"].StringValue()
			model := optionMap["model"].StringValue()

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Asking AI... `%s`", command),
				},
			})
			message, err := s.InteractionResponse(i.Interaction)
			if err != nil {
				log.Println("Error:", err)
			} else {
				go RespondWithGeneratedContent(message, command, model)
			}
		},
	}
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})
	err := s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	log.Println("Checking for available ollama models...")

	models, err := api.OllamaModelNames()
	if err != nil || len(models) == 0 {
		log.Fatalf("Cannot get models: %v", err)
		return
	}

	choices := make([]*discordgo.ApplicationCommandOptionChoice, len(models))

	for i, v := range models {
		choices[i] = &discordgo.ApplicationCommandOptionChoice{
			Name:  v,
			Value: v,
		}
	}

	log.Println("Adding commands...")

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "generate",
			Description: "Generates a response from the AI",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "model",
					Description: "LLM Model",
					Choices:     choices,
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "prompt",
					Description: "LLM Prompt",
					Required:    true,
				},
				// {
				// 	Type:        discordgo.ApplicationCommandOptionAttachment,
				// 	Name:        "attachment",
				// 	Description: "Attachment to be used as a prompt",
				// 	Required:    false,
				// },
			},
		},
	}

	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, *GuildID, v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	if *RemoveCommands {
		log.Println("Removing commands...")

		for _, v := range registeredCommands {
			err := s.ApplicationCommandDelete(s.State.User.ID, *GuildID, v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
	}

	log.Println("Gracefully shutting down.")
}

func RespondWithGeneratedContent(message *discordgo.Message, command string, model string) {
	reference := message.Reference()

	response, err := api.Generate(command, model)
	if err != nil {
		log.Println("Error:", err)
		return
	}

	generateResponse := ollama.GenerateResponse{}

	err = json.Unmarshal([]byte(response), &generateResponse)
	if err != nil {
		log.Println("Error:", err)
		return
	}

	// if the response is more than 2000 characters, split it into multiple messages (discord limit)
	if len(generateResponse.Response) > 2000 {
		var response string
		for i := 0; i < len(generateResponse.Response); i += 2000 {
			if i+2000 < len(generateResponse.Response) {
				response = generateResponse.Response[i : i+2000]
			} else {
				response = generateResponse.Response[i:]
			}
			_, err = s.ChannelMessageSendReply(message.ChannelID, response, reference)
			if err != nil {
				log.Println("Error:", err)
				return
			}
		}
	} else {
		_, err = s.ChannelMessageSendReply(message.ChannelID, generateResponse.Response, reference)
		if err != nil {
			log.Println("Error:", err)
			return
		}
	}
}
