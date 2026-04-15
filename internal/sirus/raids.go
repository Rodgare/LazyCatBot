package sirus

import "fmt"

type RaidKey struct {
	ID         int
	Difficulty int
}

var ShortRaidNames = map[RaidKey]string{
	// Бронзовое святилище
	{ID: 734, Difficulty: 1}: "БС",

	// Крепость Бурь (Око)
	{ID: 550, Difficulty: 1}: "Око об",
	{ID: 550, Difficulty: 3}: "Око хм",

	// Змеиное святилище
	{ID: 548, Difficulty: 1}: "ЗС об",

	// Тол'Гародская тюрьма
	{ID: 838, Difficulty: 0}: "ТТГ об",
	{ID: 838, Difficulty: 1}: "ТТГ хм",

	// Рубиновое святилище
	{ID: 724, Difficulty: 0}: "РС10 об",
	{ID: 724, Difficulty: 1}: "РС25 об",
	{ID: 724, Difficulty: 2}: "РС10 хм",
	{ID: 724, Difficulty: 3}: "РС25 хм",

	// Цитадель Ледяной Короны
	{ID: 631, Difficulty: 0}: "ЦЛК10 об",
	{ID: 631, Difficulty: 1}: "ЦЛК25 об",
	{ID: 631, Difficulty: 2}: "ЦЛК10 хм",
	{ID: 631, Difficulty: 3}: "ЦЛК25 хм",

	// Зул'Аман
	{ID: 568, Difficulty: 0}: "ЗА",

	// Склеп Аркавона
	{ID: 624, Difficulty: 0}: "СА10",
	{ID: 624, Difficulty: 1}: "СА25",

	// Логово Груула
	{ID: 565, Difficulty: 1}: "Груул об",
	{ID: 565, Difficulty: 3}: "Груул хм",

	// Логово Магтеридона
	{ID: 544, Difficulty: 1}: "Магик об",
	{ID: 544, Difficulty: 3}: "Магик хм",

	// Каражан
	{ID: 532, Difficulty: 0}: "Кара об",
	{ID: 532, Difficulty: 2}: "Кара хм",

	// Испытание Крестоносца
	{ID: 649, Difficulty: 0}: "ИК10",
	{ID: 649, Difficulty: 1}: "ИК25",
	{ID: 649, Difficulty: 2}: "ИВК10",
	{ID: 649, Difficulty: 3}: "ИВК25",

	// Логово Ониксии
	{ID: 249, Difficulty: 0}: "Оня10",
	{ID: 249, Difficulty: 1}: "Оня25",

	// Ульдуар
	{ID: 603, Difficulty: 0}: "Ульда10",
	{ID: 603, Difficulty: 1}: "Ульда25",

	// Наксрамас
	{ID: 533, Difficulty: 2}: "Накс10",
	{ID: 533, Difficulty: 3}: "Накс25",

	// Око Вечности
	{ID: 616, Difficulty: 0}: "Око10",
	{ID: 616, Difficulty: 1}: "Око25",
}

func GetRaidName(id int, diff int) (string, error) {
	key := RaidKey{ID: id, Difficulty: diff}

	if shortName, ok := ShortRaidNames[key]; ok {
		return shortName, nil
	}

	return "", fmt.Errorf("unknown raid: ID %d, Diff %d", id, diff)
}