package application

import (
	"context"
	"encoding/json"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/hongliang5316/midjourney-apiserver/internal/service"
	"github.com/hongliang5316/midjourney-apiserver/pkg/store"
)

var (
	m1 = regexp.MustCompile(` --seed [\d\w]+`)
)

func (app *Application) handleDescribeUpdateEvent(m *discordgo.MessageUpdate) {
	service.DescribeInfoCh <- *m.Embeds[0]
}

// maybe useful in the future
func (app *Application) handleDescribeEvent(m *discordgo.MessageCreate) {
}

func (app *Application) handleRateEvent(m *discordgo.MessageUpdate) {
	c := newContent(m.Content)
	rate := c.getProcessRate()
	if rate == "" {
		return
	}

	log.Printf("update process rate: %s, %s", m.ID, rate)
	if err := app.Store.UpdateProcessRate(context.Background(), m.ID, rate); err != nil {
		log.Printf("Call store.UpdateProcessRate failed, err: %+v", err)
	}
}

func (app *Application) handleCompleteEvent(m *discordgo.MessageCreate) {
	c := newContent(m.Content)
	mode := c.getMode()
	prompt := c.getPrompt()

	// promptWithSeed := m1.FindString(prompt)
	// if promptWithSeed == "" {
	// 	if m.MessageReference != nil {
	// 		meta, err := app.Store.GetMetaData(context.Background(), m.MessageReference.MessageID)
	// 		if err != nil {
	// 			log.Printf("err getting message id: %s", err)
	// 			return
	// 		}
	// 		prompt = meta.Prompt
	// 	}
	// }

	log.Printf("mode: %s, prompt: %s", mode, prompt)

	if err := app.Store.SaveWithComplete(context.Background(), m.ID, prompt, mode, toJson(m.Attachments), webhookCallback); err != nil {
		log.Printf("Call store.SaveWithComplete failed, err: %+v", err)
		return
	}
}

func (app *Application) handleEmbedErrorEvent(m *discordgo.MessageCreate) {
	e := m.Embeds[0]
	prefix := "/imagine "
	if !strings.HasPrefix(e.Footer.Text, prefix) {
		return
	}

	prompt := strings.Replace(e.Footer.Text, prefix, "", 1)
	key := store.GetKey(prompt)

	ch := service.KeyChan.Get(key)
	if ch == nil { // timeout or other exception
		return
	}

	if e.Title == "Job queued" {
		log.Printf("Job queued, key: %s, len: %d", key, len(key))
		log.Printf("save meta: %s, %s", m.ID, prompt)
		startTime := time.Now().Unix()
		if err := app.Store.SaveMeta(
			context.Background(),
			m.ID,
			prompt,
			store.StatusJobQueued,
			store.TypeImagine,
			startTime,
		); err != nil {
			log.Printf("Call store.SaveMeta failed, err: %+v", err)
			return
		}

		ch <- service.MessageInfo{
			ID:        m.ID,
			StartTime: startTime,
			Error:     nil,
		}

		return
	}

	ch <- service.MessageInfo{
		ID:        m.ID,
		StartTime: time.Now().Unix(),
		Error:     e,
	}
}

func (app *Application) handleWaitingToStartEvent(m *discordgo.MessageCreate) {
	c := newContent(m.Content)
	prompt := c.getPrompt()

	log.Printf("my id: %s", m.ID)
	if m.MessageReference != nil {
		log.Printf("message reference id: %s", m.MessageReference.MessageID)
	}

	// promptWithSeed := m1.FindString(prompt)
	// if promptWithSeed == "" {
	// 	if m.MessageReference != nil {
	// 		meta, err := app.Store.GetMetaData(context.Background(), m.MessageReference.MessageID)
	// 		if err != nil {
	// 			log.Printf("err getting message id: %s", err)
	// 			return
	// 		}
	// 		prompt = meta.Prompt
	// 	}
	// }

	key := store.GetKey(prompt)
	ch := service.KeyChan.Get(key)
	if ch == nil { // timeout or other exception
		return
	}

	typ := store.TypeImagine
	if strings.HasPrefix(m.Content, "Upscaling image") {
		typ = store.TypeUpscale
	}

	log.Printf("Waiting, type: %s, key: %s, len: %d", typ, key, len(key))

	startTime := time.Now().Unix()
	if err := app.Store.SaveMeta(
		context.Background(),
		m.ID,
		prompt,
		store.StatusWaitingToStart,
		typ,
		startTime,
	); err != nil {
		log.Printf("Call store.SaveMeta failed, err: %+v", err)
		return
	}

	ch <- service.MessageInfo{
		ID:        m.ID,
		StartTime: startTime,
		Error:     nil,
	}
}

func toJson(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
