package configs_export

import (
	"fmt"

	"github.com/darklab8/fl-configs/configs/cfgtype"
	"github.com/darklab8/fl-configs/configs/configs_mapped/freelancer_mapped/data_mapped/initialworld/flhash"
)

func (e *Exporter) findable_in_loot() map[string]bool {
	if e.findable_in_loot_cache != nil {
		return e.findable_in_loot_cache
	}

	e.findable_in_loot_cache = make(map[string]bool)

	for _, system := range e.Configs.Systems.Systems {
		for _, wreck := range system.Wrecks {
			louadout_nickname := wreck.Loadout.Get()
			if loadout, ok := e.Configs.Loadouts.LoadoutsByNick[louadout_nickname]; ok {
				for _, cargo := range loadout.Cargos {
					e.findable_in_loot_cache[cargo.Nickname.Get()] = true
				}
			}
		}
	}

	for _, npc_arch := range e.Configs.NpcShips.NpcShips {
		loadout_nickname := npc_arch.Loadout.Get()
		if loadout, ok := e.Configs.Loadouts.LoadoutsByNick[loadout_nickname]; ok {
			for _, cargo := range loadout.Cargos {
				e.findable_in_loot_cache[cargo.Nickname.Get()] = true
			}
		}
	}
	return e.findable_in_loot_cache
}

/*
It fixes issue of Guns obtainable only via wrecks being invisible
*/
const (
	BaseLootableName     = "Lootable"
	BaseLootableFaction  = "Wrecks and Missions"
	BaseLootableNickname = "base_loots"
)

func (e *Exporter) EnhanceBasesWithLoot(bases []*Base) []*Base {

	in_wrecks := e.findable_in_loot()

	base := &Base{
		Name:               "Lootable",
		MarketGoodsPerNick: make(map[CommodityKey]MarketGood),
		Nickname:           cfgtype.BaseUniNick(BaseLootableNickname),
		InfocardKey:        InfocardKey(BaseLootableNickname),
		SystemNickname:     "neverwhere",
		System:             "Neverwhere",
		Region:             "Neverwhere",
		FactionName:        BaseLootableFaction,
	}

	base.Archetypes = append(base.Archetypes, BaseLootableNickname)

	for wreck, _ := range in_wrecks {
		market_good := MarketGood{
			Nickname:             wreck,
			NicknameHash:         flhash.HashNickname(wreck),
			Infocard:             InfocardKey(wreck),
			BaseSells:            true,
			Type:                 "lootable",
			ShipClass:            -1,
			IsServerSideOverride: true,
		}
		e.Hashes[market_good.Nickname] = market_good.NicknameHash

		if good, found_good := e.Configs.Goods.GoodsMap[market_good.Nickname]; found_good {
			category := good.Category.Get()
			market_good.Type = fmt.Sprintf("%s loot", category)
			if equip, ok := e.Configs.Equip.ItemsMap[market_good.Nickname]; ok {
				market_good.Type = fmt.Sprintf("%s loot", equip.Category)
				e.exportInfocards(InfocardKey(market_good.Nickname), equip.IdsInfo.Get())
			}

		}
		if equip, ok := e.Configs.Equip.ItemsMap[wreck]; ok {
			market_good.Name = e.GetInfocardName(equip.IdsName.Get(), wreck)
			e.exportInfocards(InfocardKey(market_good.Nickname), equip.IdsInfo.Get())
		}

		market_good_key := GetCommodityKey(market_good.Nickname, market_good.ShipClass)
		base.MarketGoodsPerNick[market_good_key] = market_good
	}

	var sb []InfocardLine
	sb = append(sb, NewInfocardSimpleLine(base.Name))
	sb = append(sb, NewInfocardSimpleLine(`This is only pseudo base to show availability of lootable content`))
	sb = append(sb, NewInfocardSimpleLine(`The content is findable in wrecks or drops from ships at missions`))

	e.Infocards[InfocardKey(base.Nickname)] = sb

	base.Infocard = sb

	bases = append(bases, base)
	return bases
}
