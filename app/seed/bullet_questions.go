package seed

import (
	"encoding/json"
	"fmt"

	"house-hunt/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const seedUserID = "00000000-0000-0000-0000-000000000001"

type bulletSeedAnswer struct {
	Label   string
	Aliases []string
}

type bulletSeedQuestion struct {
	ID          string
	Prompt      string
	Explanation string
	Difficulty  int
	Answers     []bulletSeedAnswer
}

type bulletAnswerData struct {
	Question string              `json:"question"`
	Answers  []string            `json:"answers"`
	Aliases  map[string][]string `json:"aliases,omitempty"`
}

// EnsureBulletQuestions installs a small, deterministic development bank.
// The fixed IDs and ON CONFLICT behavior make repeated container starts safe.
func EnsureBulletQuestions(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		seedUser := model.User{
			ID:       seedUserID,
			Username: "brain-survival-seed",
			Email:    "seed@brain-survival.local",
			Provider: "system",
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&seedUser).Error; err != nil {
			return fmt.Errorf("create seed user: %w", err)
		}

		questions, err := buildBulletQuestionModels()
		if err != nil {
			return err
		}
		for index := range questions {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&questions[index]).Error; err != nil {
				return fmt.Errorf("create bullet seed question %s: %w", questions[index].ID, err)
			}
		}
		return nil
	})
}

func buildBulletQuestionModels() ([]model.Question, error) {
	models := make([]model.Question, 0, len(bulletSeedQuestions))
	for _, fixture := range bulletSeedQuestions {
		answerData := bulletAnswerData{
			Question: fixture.Prompt,
			Answers:  make([]string, 0, len(fixture.Answers)),
			Aliases:  make(map[string][]string),
		}
		for _, answer := range fixture.Answers {
			answerData.Answers = append(answerData.Answers, answer.Label)
			if len(answer.Aliases) > 0 {
				answerData.Aliases[answer.Label] = append([]string(nil), answer.Aliases...)
			}
		}
		encoded, err := json.Marshal(answerData)
		if err != nil {
			return nil, fmt.Errorf("encode bullet seed question %s: %w", fixture.ID, err)
		}
		models = append(models, model.Question{
			ID:            fixture.ID,
			CreatorUserID: seedUserID,
			Body:          fixture.Prompt,
			AnswerData:    string(encoded),
			Explanation:   fixture.Explanation,
			GameMode:      "bullet",
			Difficulty:    fixture.Difficulty,
			Status:        "approved",
		})
	}
	return models, nil
}

var bulletSeedQuestions = []bulletSeedQuestion{
	{
		ID:          "00000000-0000-0000-0000-000000000101",
		Prompt:      "十二支を10個答えろ！",
		Explanation: "子・丑・寅・卯・辰・巳・午・未・申・酉・戌・亥の十二支から10個答える問題です。",
		Difficulty:  1,
		Answers: []bulletSeedAnswer{
			{Label: "子", Aliases: []string{"ね", "鼠", "ねずみ"}},
			{Label: "丑", Aliases: []string{"うし", "牛"}},
			{Label: "寅", Aliases: []string{"とら", "虎"}},
			{Label: "卯", Aliases: []string{"う", "兎", "うさぎ"}},
			{Label: "辰", Aliases: []string{"たつ", "竜", "龍"}},
			{Label: "巳", Aliases: []string{"み", "蛇", "へび"}},
			{Label: "午", Aliases: []string{"うま", "馬"}},
			{Label: "未", Aliases: []string{"ひつじ", "羊"}},
			{Label: "申", Aliases: []string{"さる", "猿"}},
			{Label: "酉", Aliases: []string{"とり", "鶏", "にわとり"}},
			{Label: "戌", Aliases: []string{"いぬ", "犬"}},
			{Label: "亥", Aliases: []string{"い", "猪", "いのしし"}},
		},
	},
	{
		ID:          "00000000-0000-0000-0000-000000000102",
		Prompt:      "12星座を10個答えろ！",
		Explanation: "星占いで使われる黄道十二星座から10個答える問題です。",
		Difficulty:  2,
		Answers: []bulletSeedAnswer{
			{Label: "牡羊座", Aliases: []string{"おひつじ座", "Aries"}},
			{Label: "牡牛座", Aliases: []string{"おうし座", "Taurus"}},
			{Label: "双子座", Aliases: []string{"ふたご座", "Gemini"}},
			{Label: "蟹座", Aliases: []string{"かに座", "Cancer"}},
			{Label: "獅子座", Aliases: []string{"しし座", "Leo"}},
			{Label: "乙女座", Aliases: []string{"おとめ座", "Virgo"}},
			{Label: "天秤座", Aliases: []string{"てんびん座", "Libra"}},
			{Label: "蠍座", Aliases: []string{"さそり座", "Scorpio"}},
			{Label: "射手座", Aliases: []string{"いて座", "Sagittarius"}},
			{Label: "山羊座", Aliases: []string{"やぎ座", "Capricorn"}},
			{Label: "水瓶座", Aliases: []string{"みずがめ座", "Aquarius"}},
			{Label: "魚座", Aliases: []string{"うお座", "Pisces"}},
		},
	},
	{
		ID:          "00000000-0000-0000-0000-000000000103",
		Prompt:      "ギリシャ文字の先頭10字（α〜κ）の名称を10個答えろ！",
		Explanation: "αからκまでのギリシャ文字名を答える問題です。",
		Difficulty:  3,
		Answers: []bulletSeedAnswer{
			{Label: "アルファ", Aliases: []string{"α", "alpha"}},
			{Label: "ベータ", Aliases: []string{"β", "beta"}},
			{Label: "ガンマ", Aliases: []string{"γ", "gamma"}},
			{Label: "デルタ", Aliases: []string{"δ", "delta"}},
			{Label: "イプシロン", Aliases: []string{"エプシロン", "ε", "epsilon"}},
			{Label: "ゼータ", Aliases: []string{"ツェータ", "ζ", "zeta"}},
			{Label: "イータ", Aliases: []string{"エータ", "η", "eta"}},
			{Label: "シータ", Aliases: []string{"テータ", "θ", "theta"}},
			{Label: "イオタ", Aliases: []string{"ι", "iota"}},
			{Label: "カッパ", Aliases: []string{"κ", "kappa"}},
		},
	},
	{
		ID:          "00000000-0000-0000-0000-000000000104",
		Prompt:      "旧暦1月から12月までの代表的な和風月名を10個答えろ！",
		Explanation: "睦月から師走までの和風月名を答える問題です。",
		Difficulty:  3,
		Answers: []bulletSeedAnswer{
			{Label: "睦月", Aliases: []string{"むつき"}},
			{Label: "如月", Aliases: []string{"きさらぎ", "衣更着"}},
			{Label: "弥生", Aliases: []string{"やよい"}},
			{Label: "卯月", Aliases: []string{"うづき"}},
			{Label: "皐月", Aliases: []string{"さつき", "早月"}},
			{Label: "水無月", Aliases: []string{"みなづき", "みなつき"}},
			{Label: "文月", Aliases: []string{"ふみづき", "ふづき"}},
			{Label: "葉月", Aliases: []string{"はづき", "はつき"}},
			{Label: "長月", Aliases: []string{"ながつき", "ながづき"}},
			{Label: "神無月", Aliases: []string{"かんなづき", "かみなづき", "神在月"}},
			{Label: "霜月", Aliases: []string{"しもつき"}},
			{Label: "師走", Aliases: []string{"しわす"}},
		},
	},
	{
		ID:          "00000000-0000-0000-0000-000000000105",
		Prompt:      "原子番号1番から10番までの元素を10個答えろ！",
		Explanation: "水素からネオンまで、原子番号1〜10の元素を答える問題です。",
		Difficulty:  3,
		Answers: []bulletSeedAnswer{
			{Label: "水素", Aliases: []string{"すいそ", "H", "hydrogen"}},
			{Label: "ヘリウム", Aliases: []string{"He", "helium"}},
			{Label: "リチウム", Aliases: []string{"Li", "lithium"}},
			{Label: "ベリリウム", Aliases: []string{"Be", "beryllium"}},
			{Label: "ホウ素", Aliases: []string{"ほうそ", "硼素", "B", "boron"}},
			{Label: "炭素", Aliases: []string{"たんそ", "C", "carbon"}},
			{Label: "窒素", Aliases: []string{"ちっそ", "N", "nitrogen"}},
			{Label: "酸素", Aliases: []string{"さんそ", "O", "oxygen"}},
			{Label: "フッ素", Aliases: []string{"ふっそ", "弗素", "F", "fluorine"}},
			{Label: "ネオン", Aliases: []string{"Ne", "neon"}},
		},
	},
	{
		ID:          "00000000-0000-0000-0000-000000000106",
		Prompt:      "辺が3本から12本までの多角形の名称を10個答えろ！",
		Explanation: "三角形から十二角形までを答える問題です。",
		Difficulty:  4,
		Answers: []bulletSeedAnswer{
			{Label: "三角形", Aliases: []string{"さんかくけい", "3角形", "triangle"}},
			{Label: "四角形", Aliases: []string{"しかくけい", "4角形", "quadrilateral"}},
			{Label: "五角形", Aliases: []string{"ごかくけい", "5角形", "pentagon"}},
			{Label: "六角形", Aliases: []string{"ろっかくけい", "6角形", "hexagon"}},
			{Label: "七角形", Aliases: []string{"ななかくけい", "7角形", "heptagon"}},
			{Label: "八角形", Aliases: []string{"はっかくけい", "8角形", "octagon"}},
			{Label: "九角形", Aliases: []string{"きゅうかくけい", "9角形", "nonagon"}},
			{Label: "十角形", Aliases: []string{"じゅっかくけい", "10角形", "decagon"}},
			{Label: "十一角形", Aliases: []string{"じゅういちかくけい", "11角形", "undecagon"}},
			{Label: "十二角形", Aliases: []string{"じゅうにかくけい", "12角形", "dodecagon"}},
		},
	},
	{
		ID:          "00000000-0000-0000-0000-000000000107",
		Prompt:      "南アメリカにある12の独立国を10個答えろ！",
		Explanation: "南アメリカ大陸の独立国から10か国を答える問題です。",
		Difficulty:  4,
		Answers: []bulletSeedAnswer{
			{Label: "アルゼンチン", Aliases: []string{"Argentina"}},
			{Label: "ボリビア", Aliases: []string{"Bolivia"}},
			{Label: "ブラジル", Aliases: []string{"Brazil"}},
			{Label: "チリ", Aliases: []string{"Chile"}},
			{Label: "コロンビア", Aliases: []string{"Colombia"}},
			{Label: "エクアドル", Aliases: []string{"Ecuador"}},
			{Label: "ガイアナ", Aliases: []string{"Guyana"}},
			{Label: "パラグアイ", Aliases: []string{"Paraguay"}},
			{Label: "ペルー", Aliases: []string{"Peru"}},
			{Label: "スリナム", Aliases: []string{"Suriname"}},
			{Label: "ウルグアイ", Aliases: []string{"Uruguay"}},
			{Label: "ベネズエラ", Aliases: []string{"Venezuela"}},
		},
	},
}
