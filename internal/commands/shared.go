package commands

import (
	"errors"
	"strings"

	"kurohelper/internal/store"
	"kurohelper/internal/utils"
	kurohelperdb "kurohelperservice/db"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

type CharacterData struct {
	Name string
	Role string
}

const (
	Played = 1 << iota
	Wish
)

const (
	PlaceholderImageURL = "https://cdn.kurohelper.com/docs/neneGIF.gif"
)

// 資料分頁
func Pagination[T any](result *[]T, page int, useCache bool) bool {
	resultLen := len(*result)
	expectedMin := page * 10
	expectedMax := page*10 + 10

	if !useCache || page == 0 {
		if resultLen > 10 {
			*result = (*result)[:10]
			return true
		}
		return false
	} else {
		if resultLen > expectedMax {
			*result = (*result)[expectedMin:expectedMax]
			return true
		} else {
			*result = (*result)[expectedMin:]
			return false
		}
	}
}

// 資料分頁(回傳切片本身版本)
func PaginationR[T any](result []T, page int, useCache bool) ([]T, bool) {
	resultLen := len(result)
	expectedMin := page * 10
	expectedMax := page*10 + 10

	if !useCache || page == 0 {
		if resultLen > 10 {
			return result[:10], true
		}
		return result, false
	} else {
		if resultLen > expectedMax {
			return result[expectedMin:expectedMax], true
		} else {
			return result[min(expectedMin, resultLen):], false
		}
	}
}

// 產生顯示圖片，會檢查白名單來判斷要不要顯示
func GenerateImage(i *discordgo.InteractionCreate, url string) *discordgo.MessageEmbedImage {
	var image *discordgo.MessageEmbedImage
	if i.GuildID != "" {
		// guild
		if _, ok := store.GuildDiscordAllowList[i.GuildID]; ok {
			image = &discordgo.MessageEmbedImage{
				URL: url,
			}
		}
	} else {
		// DM
		userID := utils.GetUserID(i)
		if _, ok := store.GuildDiscordAllowList[userID]; ok {
			image = &discordgo.MessageEmbedImage{
				URL: url,
			}
		}
	}
	return image
}

// LoadGameStateMaps 取得使用者的遊玩狀態與願望清單，並以 GameErogsID 建立索引
func LoadGameStateMaps(discordID string) (statusMap map[int]kurohelperdb.UserGameStatus, inWishMap map[int]struct{}, err error) {
	statusMap = make(map[int]kurohelperdb.UserGameStatus)
	inWishMap = make(map[int]struct{})

	if strings.TrimSpace(discordID) == "" {
		return statusMap, inWishMap, nil
	}
	if _, ok := store.UserStore[discordID]; !ok {
		return statusMap, inWishMap, nil
	}

	userGames, err := kurohelperdb.GetUserGameByDiscordID(kurohelperdb.Dbs, discordID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return statusMap, inWishMap, nil
	}
	if err != nil {
		return nil, nil, err
	}

	for _, item := range userGames {
		if item.Status != kurohelperdb.UserGameStatusNone {
			statusMap[item.GameErogsID] = item.Status
		}
		if item.WishListMark {
			inWishMap[item.GameErogsID] = struct{}{}
		}
	}

	return statusMap, inWishMap, nil
}

// FormatGameFlags 將遊玩狀態與願望清單轉換成 Discord 顯示圖示
func FormatGameFlags(status kurohelperdb.UserGameStatus, inWish bool) string {
	flags := make([]string, 0, 2)
	switch status {
	case kurohelperdb.UserGameStatusFinished:
		flags = append(flags, "✅")
	case kurohelperdb.UserGameStatusPlaying:
		flags = append(flags, "🎮")
	case kurohelperdb.UserGameStatusStalled:
		flags = append(flags, "⏸️")
	case kurohelperdb.UserGameStatusDropped:
		flags = append(flags, "🗑️")
	}
	if inWish {
		flags = append(flags, "❤️")
	}
	return strings.Join(flags, " ")
}
