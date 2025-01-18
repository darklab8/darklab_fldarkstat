package configs_export

import (
	"fmt"

	"github.com/darklab8/fl-configs/configs/cfgtype"
	"github.com/darklab8/fl-configs/configs/configs_mapped/freelancer_mapped/data_mapped/initialworld/flhash"
	"github.com/darklab8/fl-configs/configs/configs_mapped/freelancer_mapped/data_mapped/universe_mapped"
)

type GoodAtBase struct {
	BaseNickname      cfgtype.BaseUniNick
	BaseSells         bool
	PriceBaseBuysFor  int
	PriceBaseSellsFor int
	Volume            float64
	ShipClass         cfgtype.ShipClass
	LevelRequired     int
	RepRequired       float64

	NotBuyable           bool
	IsServerSideOverride bool

	IsTransportUnreachable bool

	BaseInfo
}

type Commodity struct {
	Nickname              string
	NicknameHash          flhash.HashCode
	Name                  string
	Combinable            bool
	Volume                float64
	ShipClass             cfgtype.ShipClass
	NameID                int
	InfocardID            int
	Infocard              InfocardKey
	Bases                 map[cfgtype.BaseUniNick]*GoodAtBase
	PriceBestBaseBuysFor  int
	PriceBestBaseSellsFor int
	ProffitMargin         int
	baseAllTradeRoutes
	Mass float64
}

func GetPricePerVoume(price int, volume float64) float64 {
	if volume == 0 {
		return -1
	}
	return float64(price) / float64(volume)
}

func (e *Exporter) GetCommodities() []*Commodity {
	commodities := make([]*Commodity, 0, 100)

	for _, comm := range e.Configs.Goods.Commodities {
		equipment_name := comm.Equipment.Get()
		equipment := e.Configs.Equip.CommoditiesMap[equipment_name]

		for _, volume_info := range equipment.Volumes {
			commodity := &Commodity{
				Bases: make(map[cfgtype.BaseUniNick]*GoodAtBase),
			}
			commodity.Mass, _ = equipment.Mass.GetValue()

			commodity.Nickname = comm.Nickname.Get()
			commodity.NicknameHash = flhash.HashNickname(commodity.Nickname)
			e.Hashes[commodity.Nickname] = commodity.NicknameHash

			commodity.Combinable = comm.Combinable.Get()

			commodity.NameID = equipment.IdsName.Get()

			commodity.Name = e.GetInfocardName(equipment.IdsName.Get(), commodity.Nickname)
			e.exportInfocards(commodity.Infocard, equipment.IdsInfo.Get())
			commodity.InfocardID = equipment.IdsInfo.Get()

			commodity.Volume = volume_info.Volume.Get()
			commodity.ShipClass = volume_info.GetShipClass()
			commodity.Infocard = InfocardKey(commodity.Nickname)

			base_item_price := comm.Price.Get()

			commodity.Bases = e.GetAtBasesSold(GetCommodityAtBasesInput{
				Nickname:  commodity.Nickname,
				Price:     base_item_price,
				Volume:    commodity.Volume,
				ShipClass: commodity.ShipClass,
			})

			for _, base_info := range commodity.Bases {
				if base_info.PriceBaseBuysFor > commodity.PriceBestBaseBuysFor {
					commodity.PriceBestBaseBuysFor = base_info.PriceBaseBuysFor
				}
				if base_info.PriceBaseSellsFor < commodity.PriceBestBaseSellsFor || commodity.PriceBestBaseSellsFor == 0 {
					if base_info.BaseSells && base_info.PriceBaseSellsFor > 0 {
						commodity.PriceBestBaseSellsFor = base_info.PriceBaseSellsFor
					}

				}
			}

			if commodity.PriceBestBaseBuysFor > 0 && commodity.PriceBestBaseSellsFor > 0 {
				commodity.ProffitMargin = commodity.PriceBestBaseBuysFor - commodity.PriceBestBaseSellsFor
			}

			commodities = append(commodities, commodity)
		}

	}

	return commodities
}

type GetCommodityAtBasesInput struct {
	Nickname  string
	Price     int
	Volume    float64
	ShipClass cfgtype.ShipClass
}

func (e *Exporter) ServerSideMarketGoodsOverrides(commodity GetCommodityAtBasesInput) map[cfgtype.BaseUniNick]*GoodAtBase {
	var bases_already_found map[cfgtype.BaseUniNick]*GoodAtBase = make(map[cfgtype.BaseUniNick]*GoodAtBase)

	for _, base_market := range e.Configs.Discovery.Prices.BasesPerGood[commodity.Nickname] {
		var base_info *GoodAtBase
		base_nickname := cfgtype.BaseUniNick(base_market.BaseNickname.Get())

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered in f", r)
				fmt.Println("recovered base_nickname", base_nickname)
				fmt.Println("recovered commodity nickname", commodity.Nickname)
				panic(r)
			}
		}()

		base_info = &GoodAtBase{
			NotBuyable:           false,
			BaseNickname:         base_nickname,
			BaseSells:            base_market.BaseSells.Get(),
			PriceBaseBuysFor:     base_market.PriceBaseBuysFor.Get(),
			PriceBaseSellsFor:    base_market.PriceBaseSellsFor.Get(),
			Volume:               commodity.Volume,
			ShipClass:            commodity.ShipClass,
			IsServerSideOverride: true,
		}

		base_info.BaseInfo = e.GetBaseInfo(universe_mapped.BaseNickname(base_info.BaseNickname))

		if e.useful_bases_by_nick != nil {
			if _, ok := e.useful_bases_by_nick[base_info.BaseNickname]; !ok {
				base_info.NotBuyable = true
			}
		}

		bases_already_found[base_info.BaseNickname] = base_info
	}
	return bases_already_found
}

func (e *Exporter) GetAtBasesSold(commodity GetCommodityAtBasesInput) map[cfgtype.BaseUniNick]*GoodAtBase {
	var goods_per_base map[cfgtype.BaseUniNick]*GoodAtBase = make(map[cfgtype.BaseUniNick]*GoodAtBase)

	for _, base_market := range e.Configs.Market.BasesPerGood[commodity.Nickname] {
		base_nickname := base_market.Base

		market_good := base_market.MarketGood
		base_info := &GoodAtBase{
			NotBuyable: false,
			Volume:     commodity.Volume,
			ShipClass:  commodity.ShipClass,
		}
		base_info.BaseSells = market_good.BaseSells()
		base_info.BaseNickname = base_nickname

		base_info.PriceBaseSellsFor = int(market_good.PriceModifier.Get() * float64(commodity.Price))

		if e.Configs.Discovery != nil {
			base_info.PriceBaseBuysFor = market_good.BaseSellsIPositiveAndDiscoSellPrice.Get()
		} else {
			base_info.PriceBaseBuysFor = base_info.PriceBaseSellsFor
		}

		base_info.LevelRequired = market_good.LevelRequired.Get()
		base_info.RepRequired = market_good.RepRequired.Get()

		base_info.BaseInfo = e.GetBaseInfo(universe_mapped.BaseNickname(base_info.BaseNickname))

		if e.useful_bases_by_nick != nil {
			if _, ok := e.useful_bases_by_nick[base_info.BaseNickname]; !ok {
				base_info.NotBuyable = true
			}
		}

		goods_per_base[base_info.BaseNickname] = base_info
	}

	if e.Configs.Discovery != nil {
		serverside_overrides := e.ServerSideMarketGoodsOverrides(commodity)
		for _, item := range serverside_overrides {
			goods_per_base[item.BaseNickname] = item
		}

	}
	if e.Configs.Discovery != nil || e.Configs.FLSR != nil {
		pob_produced := e.pob_produced()
		if _, ok := pob_produced[commodity.Nickname]; ok {
			good_to_add := &GoodAtBase{
				BaseNickname:         pob_crafts_nickname,
				BaseSells:            true,
				IsServerSideOverride: true,
				Volume:               commodity.Volume,
				BaseInfo: BaseInfo{
					BaseName:    e.Configs.CraftableBaseName(),
					SystemName:  "Neverwhere",
					Region:      "Neverwhere",
					FactionName: "Neverwhere",
				},
			}
			goods_per_base[pob_crafts_nickname] = good_to_add

		}
	}

	loot_findable := e.findable_in_loot()
	if _, ok := loot_findable[commodity.Nickname]; ok {
		good_to_add := &GoodAtBase{
			BaseNickname:         BaseLootableNickname,
			BaseSells:            true,
			IsServerSideOverride: false,
			Volume:               commodity.Volume,

			BaseInfo: BaseInfo{
				BaseName:    BaseLootableName,
				SystemName:  "Neverwhere",
				Region:      "Neverwhere",
				FactionName: BaseLootableFaction,
			},
		}
		goods_per_base[BaseLootableNickname] = good_to_add

	}

	if e.Configs.Discovery != nil {
		pob_buyable := e.get_pob_buyable()
		if goods, ok := pob_buyable[commodity.Nickname]; ok {
			for _, good := range goods {
				good_to_add := &GoodAtBase{
					BaseNickname:         cfgtype.BaseUniNick(good.PobNickname),
					BaseSells:            good.Quantity > good.MinStock,
					IsServerSideOverride: true,
					PriceBaseBuysFor:     good.SellPrice,
					PriceBaseSellsFor:    good.Price,
					Volume:               commodity.Volume,
					BaseInfo: BaseInfo{
						BaseName:    "(PoB) " + good.PoBName,
						SystemName:  good.SystemName,
						FactionName: good.FactionName,
					},
				}

				if good.System != nil {
					good_to_add.BaseInfo.Region = e.GetRegionName(good.System)
				}
				if good.BasePos != nil && good.System != nil {
					good_to_add.BasePos = *good.BasePos
					good_to_add.SectorCoord = VectorToSectorCoord(good.System, *good.BasePos)
				}
				goods_per_base[cfgtype.BaseUniNick(good.PobNickname)] = good_to_add
			}
		}
	}

	return goods_per_base
}

type BaseInfo struct {
	BaseName    string
	SystemName  string
	Region      string
	FactionName string
	BasePos     cfgtype.Vector
	SectorCoord string
}

func (e *Exporter) GetRegionName(system *universe_mapped.System) string {
	return e.Configs.GetRegionName(system)
}

func (e *Exporter) GetBaseInfo(base_nickname universe_mapped.BaseNickname) BaseInfo {
	var result BaseInfo
	universe_base, found_universe_base := e.Configs.Universe.BasesMap[universe_mapped.BaseNickname(base_nickname)]

	if !found_universe_base {
		return result
	}

	result.BaseName = e.GetInfocardName(universe_base.StridName.Get(), string(base_nickname))
	system_nickname := universe_base.System.Get()

	system, system_ok := e.Configs.Universe.SystemMap[universe_mapped.SystemNickname(system_nickname)]
	if system_ok {
		result.SystemName = e.GetInfocardName(system.StridName.Get(), system_nickname)
		result.Region = e.GetRegionName(system)
	}

	var reputation_nickname string
	if system, ok := e.Configs.Systems.SystemsMap[universe_base.System.Get()]; ok {
		for _, system_base := range system.Bases {
			if system_base.IdsName.Get() != universe_base.StridName.Get() {
				continue
			}

			reputation_nickname = system_base.RepNickname.Get()
			result.BasePos = system_base.Pos.Get()
		}

	}

	result.SectorCoord = VectorToSectorCoord(system, result.BasePos)

	var factionName string
	if group, exists := e.Configs.InitialWorld.GroupsMap[reputation_nickname]; exists {
		factionName = e.GetInfocardName(group.IdsName.Get(), reputation_nickname)
	}

	result.FactionName = factionName

	return result
}

func (e *Exporter) FilterToUsefulCommodities(commodities []*Commodity) []*Commodity {
	var items []*Commodity = make([]*Commodity, 0, len(commodities))
	for _, item := range commodities {
		if !e.Buyable(item.Bases) {
			continue
		}
		items = append(items, item)
	}
	return items
}
