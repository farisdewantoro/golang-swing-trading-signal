package telegram_bot

import (
	"context"
	"fmt"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/utils"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) handleScheduler(ctx context.Context, c telebot.Context) error {

	jobs, err := t.jobService.Get(ctx, &models.GetJobParam{
		IsActive: utils.ToPointer(true),
	})
	if err != nil {
		t.logger.Error("failed to get jobs", logrus.Fields{
			"error": err,
		})
		_, err = t.telegramRateLimiter.Send(ctx, c, commonMessageInternalError)
		return err
	}

	if len(jobs) == 0 {
		_, err = t.telegramRateLimiter.Send(ctx, c, "Tidak ada job yang aktif.")
		return err
	}

	msg := strings.Builder{}
	msg.WriteString("📋 Daftar Scheduler Aktif:\n\n")
	msg.WriteString("Pilih job yang ingin kamu lihat:\n\n")
	for idx, job := range jobs {
		msg.WriteString(fmt.Sprintf("<b>%d. %s</b>\n", idx+1, job.Name))
		msg.WriteString(fmt.Sprintf("  - %s\n", job.Description))
		msg.WriteString("\n")
	}
	msg.WriteString("\n")
	msg.WriteString("<i>👉 Tekan tombol di bawah untuk lihat detail dan jalankan manual</i>\n")

	menu := &telebot.ReplyMarkup{}
	rows := []telebot.Row{}

	for _, job := range jobs {
		btn := menu.Data(job.Name, btnDetailJob.Unique, fmt.Sprintf("%d", job.ID))
		rows = append(rows, menu.Row(btn))
	}
	rows = append(rows, menu.Row(menu.Data(btnDeleteMessage.Text, btnDeleteMessage.Unique)))
	menu.Inline(rows...)

	msgExist := c.Message()
	if msgExist != nil && msgExist.Sender.ID == t.bot.Me.ID {
		_, err = t.telegramRateLimiter.Edit(ctx, c, msgExist, msg.String(), menu, telebot.ModeHTML)
		return err
	}

	_, err = t.telegramRateLimiter.Send(ctx, c, msg.String(), menu, telebot.ModeHTML)
	return err
}

func (t *TelegramBotService) handleBtnActionBackToJobList(ctx context.Context, c telebot.Context) error {
	return t.handleScheduler(ctx, c)
}

func (t *TelegramBotService) handleBtnDetailJob(ctx context.Context, c telebot.Context) error {
	jobID := c.Data()
	jobIDInt, err := strconv.Atoi(jobID)
	if err != nil {
		t.logger.Error("failed to convert job id to int", logrus.Fields{
			"error": err,
		})
		_, err = t.telegramRateLimiter.Send(ctx, c, commonMessageInternalError)
		return err
	}

	jobs, err := t.jobService.Get(ctx, &models.GetJobParam{
		IDs: []uint{uint(jobIDInt)},
		WithTaskHistory: &models.GetTaskExecutionHistoryParam{
			Limit: utils.ToPointer(5),
		},
	})
	if err != nil {
		t.logger.Error("failed to get job by id", logrus.Fields{
			"error": err,
		})
		_, err = t.telegramRateLimiter.Send(ctx, c, commonMessageInternalError)
		return err
	}

	if len(jobs) == 0 {
		_, err = t.telegramRateLimiter.Send(ctx, c, "Job tidak ditemukan.")
		return err
	}

	job := jobs[0]

	msg := strings.Builder{}
	msg.WriteString(fmt.Sprintf("%s\n\n", job.Name))
	msg.WriteString(fmt.Sprintf("🔍 %s\n\n", job.Description))

	msg.WriteString("📅 Jadwal: \n")
	if job.Schedules[0].LastExecution.Valid {
		msg.WriteString(fmt.Sprintf(" • Last Execution : %s\n", utils.PrettyDateWithIcon(utils.TimeToWIB(job.Schedules[0].LastExecution.Time))))
	} else {
		msg.WriteString(" • Last Execution : Tidak ada\n")
	}
	if job.Schedules[0].NextExecution.Valid {
		msg.WriteString(fmt.Sprintf(" • Next Execution : %s\n", utils.PrettyDateWithIcon(utils.TimeToWIB(job.Schedules[0].NextExecution.Time))))
	} else {
		msg.WriteString(" • Next Execution : Tidak ada\n")
	}

	msg.WriteString("\n")
	msg.WriteString("📜 Riwayat Eksekusi Terakhir:\n")
	for idx, history := range job.Histories {

		icon := "🟢"
		if history.Status == models.StatusRunning {
			icon = "🟡"
		} else if history.Status == models.StatusCompleted {
			icon = "🟢"
		} else if history.Status == models.StatusFailed {
			icon = "🔴"
		} else if history.Status == models.StatusTimeout {
			icon = "🟠"
		}

		if history.CreatedAt.IsZero() {
			continue
		}
		if !history.CompletedAt.Valid {
			msg.WriteString(fmt.Sprintf("%d. %s %s - %s\n", idx+1, icon, utils.TimeToWIB(history.CreatedAt).Format("01/02 15:04"), strings.ToUpper(string(history.Status))))
			continue
		}
		duration := history.CompletedAt.Time.Sub(history.StartedAt)
		msg.WriteString(fmt.Sprintf("%d. %s %s - %s (%.1fs)\n", idx+1, icon, utils.TimeToWIB(history.CreatedAt).Format("01/02 15:04"), strings.ToUpper(string(history.Status)), duration.Seconds()))
	}

	menu := &telebot.ReplyMarkup{}
	rows := []telebot.Row{}
	btnBackJobList := menu.Data(btnActionBackToJobList.Text, btnActionBackToJobList.Unique)
	btnRun := menu.Data(btnActionRunJob.Text, btnActionRunJob.Unique, fmt.Sprintf("%d", job.ID))
	rows = append(rows, menu.Row(btnRun, btnBackJobList))
	menu.Inline(rows...)

	_, err = t.telegramRateLimiter.Edit(ctx, c, c.Message(), msg.String(), menu, telebot.ModeHTML)
	return err
}

func (t *TelegramBotService) handleBtnActionRunJob(ctx context.Context, c telebot.Context) error {
	jobID := c.Data()
	jobIDInt, err := strconv.Atoi(jobID)
	if err != nil {
		t.logger.Error("failed to convert job id to int", logrus.Fields{
			"error": err,
		})
		_, err = t.telegramRateLimiter.Send(ctx, c, commonMessageInternalError)
		return err
	}

	job, err := t.jobService.Get(ctx, &models.GetJobParam{
		IDs: []uint{uint(jobIDInt)},
	})
	if err != nil {
		t.logger.Error("failed to get job by id", logrus.Fields{
			"error": err,
		})
		_, err = t.telegramRateLimiter.Send(ctx, c, commonMessageInternalError)
		return err
	}

	if len(job) == 0 {
		_, err = t.telegramRateLimiter.Send(ctx, c, "Job tidak ditemukan.")
		return err
	}

	if err := t.jobService.RunJobTask(ctx, uint(job[0].ID)); err != nil {
		t.logger.Error("failed to run job task", logrus.Fields{
			"error": err,
		})
		_, err = t.telegramRateLimiter.Send(ctx, c, commonMessageInternalError)
		return err
	}
	_, err = t.telegramRateLimiter.Edit(ctx, c, c.Message(), fmt.Sprintf(`🚀 Job “%s” sedang diproses secara manual.

<i>Silakan cek status job melalui command /scheduler dan membuka kembali detail job ini nanti.</i>	
`, job[0].Name), telebot.ModeHTML)

	return err
}
