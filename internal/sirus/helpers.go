package sirus

import "fmt"

// GetSpecName возвращает сокращенное название специализации по ID класса и спека
func GetSpecName(classID, specID int) string {
	specs := map[int]map[int]string{
		1:  {0: "Arms", 1: "Fury", 2: "Prot"},            // Warrior
		2:  {0: "Holy", 1: "Prot", 2: "Retri"},           // Paladin
		3:  {0: "BM", 1: "MM", 2: "Surv"},                // Hunter
		4:  {0: "Assa", 1: "Combat", 2: "Sub"},           // Rogue
		5:  {0: "Disc", 1: "Holy", 2: "Shadow"},          // Priest
		6:  {0: "Blood", 1: "Frost", 2: "Unholy"},        // DK
		7:  {0: "Ele", 1: "Enh", 2: "Resto"},             // Shaman
		8:  {0: "Arcane", 1: "Fire", 2: "Frost"},         // Mage
		9:  {0: "Affli", 1: "Demo", 2: "Destro"},         // Warlock
		10: {0: "Brm", 1: "MW", 2: "WW"},                 // Monk
		11: {0: "Bal", 1: "Feral", 2: "Resto", 3: "Grd"}, // Druid (3 - Guardian)
		12: {0: "Havoc", 1: "Veng"},                      // DH
	}

	if classSpecs, ok := specs[classID]; ok {
		if name, ok := classSpecs[specID]; ok {
			return name
		}
	}
	return fmt.Sprintf("Spec:%d", specID)
}

// GetZodiacName возвращает сокращенное название созвездия (опционально)
func GetZodiacName(fullName string) string {
	return fullName
}
