package protocol

import (
	"fmt"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

type RejectionCodeData struct {
	Code        uint8
	Name        string
	Label       string
	Description string
	// Metadata string
}

var rejectionTypes map[uint8]RejectionCodeData

func GetRejectionCodes() (map[uint8]RejectionCodeData, error) {
	if rejectionTypes != nil {
		return rejectionTypes, nil
	}

	load := make([]RejectionCodeData, 0)

	if err := yaml.Unmarshal([]byte(yamlRejections), &load); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal rejection codes yaml")
	}

	rejectionTypes = make(map[uint8]RejectionCodeData)
	for _, value := range load {
		rejectionTypes[value.Code] = value
	}
	return rejectionTypes, nil
}

func GetRejectionCode(code uint8) *RejectionCodeData {
	types, err := GetRejectionCodes()
	if err != nil {
		return nil
	}
	result, exists := types[code]
	if !exists {
		return nil
	}
	return &result
}

type CurrencyTypeData struct {
	Code        string
	Name        string
	Label       string
	Description string
	// Metadata    string
}

var currencyTypes map[[3]byte]CurrencyTypeData

func GetCurrencies() (map[[3]byte]CurrencyTypeData, error) {
	if currencyTypes != nil {
		return currencyTypes, nil
	}

	load := make([]CurrencyTypeData, 0)

	if err := yaml.Unmarshal([]byte(yamlCurrencies), &load); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal currencyTypes yaml")
	}

	currencyTypes = make(map[[3]byte]CurrencyTypeData)
	for _, value := range load {
		if len(value.Code) != 3 {
			return nil, fmt.Errorf("Currency type incorrect length : %s", value.Code)
		}
		var code [3]byte
		copy(code[:], []byte(value.Code))
		currencyTypes[code] = value
	}
	return currencyTypes, nil
}

func GetCurrency(cur [3]byte) *CurrencyTypeData {
	types, err := GetCurrencies()
	if err != nil {
		return nil
	}
	result, exists := types[cur]
	if !exists {
		return nil
	}
	return &result
}

type EntityTypeData struct {
	Code        string
	Name        string
	Label       string
	Description string
	// Metadata    string
}

var entityTypes map[byte]EntityTypeData

func GetEntityTypes() (map[byte]EntityTypeData, error) {
	if entityTypes != nil {
		return entityTypes, nil
	}

	load := make([]EntityTypeData, 0)

	if err := yaml.Unmarshal([]byte(yamlEntities), &load); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal entities yaml")
	}

	entityTypes = make(map[byte]EntityTypeData)
	for _, value := range load {
		if len(value.Code) > 1 {
			return nil, fmt.Errorf("Entity type too long : %s", value.Code)
		}
		entityTypes[byte(value.Code[0])] = value
	}
	return entityTypes, nil
}

func GetEntityType(ent byte) *EntityTypeData {
	types, err := GetEntityTypes()
	if err != nil {
		return nil
	}
	result, exists := types[ent]
	if !exists {
		return nil
	}
	return &result
}

type PolityType struct {
	Code        string
	Name        string
	Label       string
	Description string
	// Metadata    string
}

var polityTypes map[string]PolityType

func GetPolityTypes() (map[string]PolityType, error) {
	if polityTypes != nil {
		return polityTypes, nil
	}

	load := make([]PolityType, 0)

	if err := yaml.Unmarshal([]byte(yamlPolities), &load); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal polities yaml")
	}

	polityTypes = make(map[string]PolityType)
	for _, value := range load {
		polityTypes[value.Code] = value
	}
	return polityTypes, nil
}

func GetPolityType(pol string) *PolityType {
	types, err := GetPolityTypes()
	if err != nil {
		return nil
	}
	result, exists := types[pol]
	if !exists {
		return nil
	}
	return &result
}

type RoleType struct {
	Code        uint8
	Name        string
	Label       string
	Description string
	// Metadata    string
}

var roleTypes map[uint8]RoleType

func GetRoleTypes() (map[uint8]RoleType, error) {
	if roleTypes != nil {
		return roleTypes, nil
	}

	load := make([]RoleType, 0)

	if err := yaml.Unmarshal([]byte(yamlRoles), &load); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal roles yaml")
	}

	roleTypes = make(map[uint8]RoleType)
	for _, value := range load {
		roleTypes[value.Code] = value
	}
	return roleTypes, nil
}

func GetRoleType(role uint8) *RoleType {
	types, err := GetRoleTypes()
	if err != nil {
		return nil
	}
	result, exists := types[role]
	if !exists {
		return nil
	}
	return &result
}

// Currencies - International Organization for Standardization code for
// Currency. 3 character code.
var yamlCurrencies = `
  - code: AED
    name: UnitedArabEmiratesDirham
    label: United Arab Emirates dirham
    metadata:
      symbol: "د.إ"
      fractions: 100
      fractional_unit: Fils
      polity:
        - ARE
      monetary_authority: Central Bank of the United Arab Emirates

  - code: AFN
    name: AfghanAfghani
    label: Afghan afghani
    metadata:
      symbol: "؋"
      fractions: 100
      fractional_unit: Pul
      polity:
        - AFG
      monetary_authority: Central Bank of Afghanistan

  - code: ALL
    name: AlbanianLek
    label: Albanian lek
    metadata:
      symbol: "L"
      fractions: 100
      fractional_unit: Qindarkë
      polity:
        - ALB
      monetary_authority: Bank of Albania

  - code: AMD
    name: ArmenianDram
    label: Armenian dram
    metadata:
      symbol: "֏"
      fractions: 100
      fractional_unit: Luma
      polity:
        - ARM
      monetary_authority: Central Bank of the Republic of Armenia

  - code: ANG
    name: NetherlandsAntilleanGuilder
    label: Netherlands Antillean guilder
    metadata:
      symbol: "ƒ"
      fractions: 100
      fractional_unit: Cent
      polity:
        - SXM
        - CUW
      monetary_authority: Central Bank of Curaçao and Sint Maarten

  - code: AOA
    name: AngolanKwanza
    label: Angolan kwanza
    metadata:
      symbol: "Kz"
      fractions: 100
      fractional_unit: Cêntimo
      polity:
        - AGO
      monetary_authority: Central Bank of Angola

  - code: ARS
    name: ArgentinePeso
    label: Argentine peso
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Centavo
      polity:
        - ARG
      monetary_authority: Central Bank of Argentina

  - code: AUD
    name: AustralianDollar
    label: Australian dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - AUS
        - NRU
        - KIR
        - TUV
      monetary_authority: Reserve Bank of Australia

  - code: AWG
    name: ArubanFlorin
    label: Aruban florin
    metadata:
      symbol: "ƒ"
      fractions: 100
      fractional_unit: Cent
      polity:
        - ABW
      monetary_authority: Central Bank of Aruba

  - code: AZN
    name: AzerbaijaniManat
    label: Azerbaijani manat
    metadata:
      symbol: "₼"
      fractions: 100
      fractional_unit: Qəpik
      polity:
        - AZE
      monetary_authority: Central Bank of Azerbaijan

  - code: BAM
    name: BosniaHerzegovinaConvertibleMark
    label: Bosnia and Herzegovina convertible mark
    metadata:
      symbol: "КМ"
      fractions: 100
      fractional_unit: Fening
      polity:
        - BIH
      monetary_authority: Central Bank of Bosnia and Herzegovina

  - code: BBD
    name: BarbadianDollar
    label: Barbadian dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - BRB
      monetary_authority: Central Bank of Barbados

  - code: BDT
    name: BangladeshiTaka
    label: Bangladeshi taka
    metadata:
      symbol: "৳"
      fractions: 100
      fractional_unit: Poisha
      polity:
        - BGD
      monetary_authority: Bangladesh Bank

  - code: BGN
    name: BulgarianLev
    label: Bulgarian lev
    metadata:
      symbol: "лв"
      fractions: 100
      fractional_unit: Stotinka
      polity:
        - BGR
      monetary_authority: Bulgarian National Bank

  - code: BHD
    name: BahrainiDinar
    label: Bahraini dinar
    metadata:
      symbol: ".د.ب"
      fractions: 1000
      fractional_unit: Fils
      polity:
        - BHR
      monetary_authority: Central Bank of Bahrain

  - code: BIF
    name: BurundianFranc
    label: Burundian franc
    metadata:
      symbol: "Fr"
      fractions: 100
      fractional_unit: Centime
      polity:
        - BDI
      monetary_authority: Bank of the Republic of Burundi

  - code: BMD
    name: BermudianDollar
    label: Bermudian dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - BMU
      monetary_authority: Bermuda Monetary Authority

  - code: BND
    name: BruneiDollar
    label: Brunei dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Sen
      polity:
        - BRN
      monetary_authority: Monetary Authority of Brunei Darussalam

  - code: BOB
    name: BolivianBoliviano
    label: Bolivian boliviano
    metadata:
      symbol: "Bs."
      fractions: 100
      fractional_unit: Centavo
      polity:
        - BOL
      monetary_authority: Central Bank of Bolivia

  - code: BRL
    name: BrazilianReal
    label: Brazilian real
    metadata:
      symbol: "R$"
      fractions: 100
      fractional_unit: Centavo
      polity:
        - BRA
      monetary_authority: Central Bank of Brazil

  - code: BSD
    name: BahamianDollar
    label: Bahamian dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - BHS
      monetary_authority: Central Bank of The Bahamas

  - code: BSV
    name: Bitcoin
    label: Bitcoin SV
    metadata:
      symbol: "₿"
      fractions: 100000000
      fractional_unit: Satoshis
      monetary_authority: Satoshi Nakamoto

  - code: BTN
    name: BhutaneseNgultrum
    label: Bhutanese ngultrum
    metadata:
      symbol: "Nu."
      fractions: 100
      fractional_unit: Chetrum
      polity:
        - BTN
      monetary_authority: Royal Monetary Authority of Bhutan

  - code: BWP
    name: BotswanaPula
    label: Botswana pula
    metadata:
      symbol: "P"
      fractions: 100
      fractional_unit: Thebe
      polity:
        - BWA
      monetary_authority: Bank of Botswana

  - code: BYN
    name: BelarusianRuble
    label: Belarusian ruble
    metadata:
      symbol: "Br"
      fractions: 100
      fractional_unit: Kapyeyka
      polity:
        - BLR
      monetary_authority: National Bank of the Republic of Belarus

  - code: BZD
    name: BelizeDollar
    label: Belize dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - BLZ
      monetary_authority: Central Bank of Belize

  - code: CAD
    name: CanadianDollar
    label: Canadian dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - CAN
      monetary_authority: Bank of Canada

  - code: CDF
    name: CongoleseFranc
    label: Congolese franc
    metadata:
      symbol: "Fr"
      fractions: 100
      fractional_unit: Centime
      polity:
        - COD
      monetary_authority: Central Bank of the Congo

  - code: CHF
    name: SwissFranc
    label: Swiss franc
    metadata:
      symbol: "Fr"
      fractions: 100
      fractional_unit: Rappen
      polity:
        - CHE
        - LIE
      monetary_authority: Swiss National Bank

  - code: CLP
    name: ChileanPeso
    label: Chilean peso
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Centavo
      polity:
        - CHL
      monetary_authority: Central Bank of Chile

  - code: CNY
    name: ChineseYuan
    label: Chinese yuan
    metadata:
      symbol: "¥"
      fractions: 100
      fractional_unit: Fen
      polity:
        - CHN
      monetary_authority: People's Bank of China

  - code: COP
    name: ColombianPeso
    label: Colombian peso
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Centavo
      polity:
        - COL
      monetary_authority: Bank of the Republic

  - code: CRC
    name: CostaRicanColon
    label: Costa Rican colon
    metadata:
      symbol: "₡"
      fractions: 100
      fractional_unit: Céntimo
      polity:
        - CRI
      monetary_authority: Central Bank of Costa Rica

  - code: CUC
    name: CubanConvertiblePeso
    label: Cuban convertible peso
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Centavo
      polity:
        - CUB
      monetary_authority: Central Bank of Cuba

  - code: CUP
    name: CubanConvertiblePeso
    label: Cuban convertible peso
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Centavo
      polity:
        - CUB
      monetary_authority: Central Bank of Cuba

  - code: CVE
    name: CapeVerdeanEscudo
    label: Cape Verdean escudo
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Centavo
      polity:
        - CUB
      monetary_authority: Bank of Cape Verde

  - code: CZK
    name: CzechKoruna
    label: Czech koruna
    metadata:
      symbol: "Kč"
      fractions: 100
      fractional_unit: Haléř
      polity:
        - CZE
      monetary_authority: Czech National Bank

  - code: DJF
    name: DjiboutianFranc
    label: Djiboutian franc
    metadata:
      symbol: "Fr"
      fractions: 100
      fractional_unit: Centime
      polity:
        - DJI
      monetary_authority: Central Bank of Djibouti

  - code: DKK
    name: DanishKrone
    label: Danish krone
    metadata:
      symbol: "kr"
      fractions: 100
      fractional_unit: Øre
      polity:
        - DNK
        - FRO
        - GRL
      monetary_authority: Danmarks Nationalbank

  - code: DOP
    name: DominicanPeso
    label: Dominican peso
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Centavo
      polity:
        - DOM
      monetary_authority: Central Bank of the Dominican Republic

  - code: DZD
    name: AlgerianDinar
    label: Algerian dinar
    metadata:
      symbol: "د.ج"
      fractions: 100
      fractional_unit: Santeem
      polity:
        - DZA
      monetary_authority: Bank of Algeria

  - code: EGP
    name: EgyptianPound
    label: Egyptian pound
    metadata:
      symbol: "E£"
      fractions: 100
      fractional_unit: Piastre
      polity:
        - EGY
      monetary_authority: Central Bank of Egypt

  - code: ERN
    name: EritreanNakfa
    label: Eritrean nakfa
    metadata:
      symbol: "Nfk"
      fractions: 100
      fractional_unit: Piastre
      polity:
        - ERI
      monetary_authority: Bank of Eritrea

  - code: ETB
    name: EthiopianBirr
    label: Ethiopian birr
    metadata:
      symbol: "Br"
      fractions: 100
      fractional_unit: Santim
      polity:
        - ETH
      monetary_authority: Bank of Eritrea

  - code: EUR
    name: Euro
    label: Euro
    metadata:
      symbol: "€"
      fractions: 100
      fractional_unit: Cent
      polity:
        - EU
        - ALA
        - AND
        - AUT
        - BEL
        - CYP
        - EST
        - FIN
        - FRA
        - ATF
        - DEU
        - GRC
        - GLP
        - IRL
        - ITA
        - LVA
        - LTU
        - LUX
        - MLT
        - GUF
        - MTQ
        - MYT
        - MCO
        - MNE
        - NLD
        - PRT
        - REU
        - BLM
        - MAF
        - SPM
        - SMR
        - SVK
        - SVN
        - ESP
      monetary_authority: European Central Bank

  - code: FJD
    name: FijianDollar
    label: Fijian dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - FJI
      monetary_authority: Reserve Bank of Fiji

  - code: FKP
    name: FalklandIslandsPound
    label: Falkland Islands pound
    metadata:
      symbol: "£"
      fractions: 100
      fractional_unit: Penny
      polity:
        - FLK
      monetary_authority: Government of the Falkland Islands

  - code: GBP
    name: PoundSterling
    label: Pound sterling
    metadata:
      symbol: "£"
      fractions: 100
      fractional_unit: Penny
      polity:
        - GBR
        - IOT
        - IMN
        - JEY
        - GGY
      monetary_authority: Bank of England

  - code: GEL
    name: GeorgianLari
    label: Georgian lari
    metadata:
      symbol: "₾"
      fractions: 100
      fractional_unit: Tetri
      polity:
        - GEO
      monetary_authority: National Bank of Georgia

  - code: GHS
    name: GhanaianCedi
    label: Ghanaian cedi
    metadata:
      symbol: "₵"
      fractions: 100
      fractional_unit: Pesewa
      polity:
        - GHA
      monetary_authority: Bank of Ghana

  - code: GIP
    name: GibraltarPound
    label: Gibraltar pound
    metadata:
      symbol: "£"
      fractions: 100
      fractional_unit: Penny
      polity:
        - GIB
      monetary_authority: Government of Gibraltar

  - code: GMD
    name: GambianDalasi
    label: Gambian dalasi
    metadata:
      symbol: "D"
      fractions: 100
      fractional_unit: Butut
      polity:
        - GMB
      monetary_authority: Central Bank of The Gambia

  - code: GNF
    name: GuineanFranc
    label: Guinean franc
    metadata:
      symbol: "Fr"
      fractions: 100
      fractional_unit: Centime
      polity:
        - GNQ
      monetary_authority: Government of Gibraltar

  - code: GTQ
    name: GuatemalanQuetzal
    label: Guatemalan quetzal
    metadata:
      symbol: "Q"
      fractions: 100
      fractional_unit: Centavo
      polity:
        - GTM
      monetary_authority: Bank of Guatemala

  - code: GYD
    name: GuyaneseDollar
    label: Guyanese dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - GUY
      monetary_authority: Bank of Guyana

  - code: HKD
    name: HongKongDollar
    label: Hong Kong dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - HKG
      monetary_authority: Hong Kong Monetary Authority

  - code: HNL
    name: HonduranLempira
    label: Honduran lempira
    metadata:
      symbol: "L"
      fractions: 100
      fractional_unit: Centavo
      polity:
        - HND
      monetary_authority: Central Bank of Honduras

  - code: HRK
    name: CroatianKuna
    label: Croatian kuna
    metadata:
      symbol: "kn"
      fractions: 100
      fractional_unit: Lipa
      polity:
        - HRV
      monetary_authority: Croatian National Bank

  - code: HTG
    name: HaitianGourde
    label: Haitian gourde
    metadata:
      symbol: "G"
      fractions: 100
      fractional_unit: 	Centime
      polity:
        - HTI
      monetary_authority: Bank of the Republic of Haiti

  - code: HUF
    name: HungarianForint
    label: Hungarian forint
    metadata:
      symbol: "Ft"
      fractions: 100
      fractional_unit: 	Fillér
      polity:
        - HUN
      monetary_authority: Hungarian National Bank

  - code: IDR
    name: IndonesianRupiah
    label: Indonesian rupiah
    metadata:
      symbol: "Rp"
      fractions: 100
      fractional_unit: 	Sen
      polity:
        - IDN
      monetary_authority: Bank of Indonesia

  - code: ILS
    name: IsraeliNewShekel
    label: Israeli new shekel
    metadata:
      symbol: "₪"
      fractions: 100
      fractional_unit: 	Agora
      polity:
        - ISR
        - PSE
      monetary_authority: Bank of Israel

  - code: INR
    name: IndianRupee
    label: Indian rupee
    metadata:
      symbol: "₹"
      fractions: 100
      fractional_unit: 	Paisa
      polity:
        - IND
        - BTN
      monetary_authority: Reserve Bank of India

  - code: IQD
    name: IraqiDinar
    label: Iraqi dinar
    metadata:
      symbol: "ع.د"
      fractions: 1000
      fractional_unit: 	Fils
      polity:
        - IRQ
      monetary_authority: Central Bank of Iraq

  - code: IRR
    name: IranianRial
    label: Iranian rial
    metadata:
      symbol: "﷼"
      fractions: 100
      fractional_unit: Dinar
      polity:
        - IRN
      monetary_authority: Central Bank of Iraq

  - code: ISK
    name: IcelandicKrona
    label: Icelandic króna
    metadata:
      symbol: "kr"
      fractions: 100
      fractional_unit: Eyrir
      polity:
        - ISL
      monetary_authority: Central Bank of Iceland

  - code: JMD
    name: JamaicanDollar
    label: Jamaican dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - JAM
      monetary_authority: Bank of Jamaica

  - code: JOD
    name: JordanianDinar
    label: Jordanian dinar
    metadata:
      symbol: "د.ا"
      fractions: 100
      fractional_unit: Piastre
      polity:
        - JOR
        - PSE
      monetary_authority: Central Bank of Jordan

  - code: JPY
    name: JapaneseYen
    label: Japanese yen
    metadata:
      symbol: "¥"
      fractions: 100
      fractional_unit: Sen
      polity:
        - JPN
      monetary_authority: Bank of Japan

  - code: KES
    name: KenyanShilling
    label: Kenyan shilling
    metadata:
      symbol: "Sh"
      fractions: 100
      fractional_unit: Cent
      polity:
        - KEN
      monetary_authority: Central Bank of Kenya

  - code: KGS
    name: KyrgyzstaniSom
    label: Kyrgyzstani som
    metadata:
      symbol: "с"
      fractions: 100
      fractional_unit: Tyiyn
      polity:
        - KGZ
      monetary_authority: National Bank of the Kyrgyz Republic

  - code: KHR
    name: CambodianRiel
    label: Cambodian riel
    metadata:
      symbol: "៛"
      fractions: 100
      fractional_unit: Sen
      polity:
        - KHM
      monetary_authority: National Bank of Cambodia

  - code: KMF
    name: ComoroFranc
    label: Comoro franc
    metadata:
      symbol: "Fr"
      fractions: 100
      fractional_unit: Sen
      polity:
        - COM
      monetary_authority: Central Bank of the Comoros

  - code: KPW
    name: NorthKoreanWon
    label: North Korean won
    metadata:
      symbol: "₩"
      fractions: 100
      fractional_unit: Chon
      polity:
        - PRK
      monetary_authority: Central Bank of the Democratic People's Republic of Korea

  - code: KRW
    name: SouthKoreanWon
    label: South Korean won
    metadata:
      symbol: "₩"
      fractions: 100
      fractional_unit: Jeon
      polity:
        - KOR
      monetary_authority: Bank of Korea

  - code: KWD
    name: KuwaitiDinar
    label: Kuwaiti dinar
    metadata:
      symbol: "د.ك"
      fractions: 1000
      fractional_unit: Fils
      polity:
        - KWT
      monetary_authority: Central Bank of Kuwait

  - code: KYD
    name: CaymanIslandsDollar
    label: Cayman Islands dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - CYM
      monetary_authority: Cayman Islands Monetary Authority

  - code: KZT
    name: KazakhstaniTenge
    label: Kazakhstani tenge
    metadata:
      symbol: "₸"
      fractions: 100
      fractional_unit: Tïın
      polity:
        - KAZ
      monetary_authority: National Bank of Kazakhstan

  - code: LAK
    name: LaoKip
    label: Lao kip
    metadata:
      symbol: "₭"
      fractions: 100
      fractional_unit: Att
      polity:
        - LAO
      monetary_authority: Bank of the Lao People's Democratic Republic

  - code: LBP
    name: LebanesePound
    label: Lebanese pound
    metadata:
      symbol: "ل.ل"
      fractions: 100
      fractional_unit: Piastre
      polity:
        - LBN
      monetary_authority: Banque du Liban

  - code: LKR
    name: SriLankanRupee
    label: Sri Lankan rupee
    metadata:
      symbol: "Rs"
      fractions: 100
      fractional_unit: Cent
      polity:
        - LKA
      monetary_authority: Central Bank of Sri Lanka

  - code: LRD
    name: LiberianDollar
    label: Liberian dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - LBR
      monetary_authority: Central Bank of Liberia

  - code: LSL
    name: LesothoLoti
    label: Lesotho loti
    metadata:
      symbol: "L"
      fractions: 100
      fractional_unit: Sente
      polity:
        - LBR
      monetary_authority: Central Bank of Lesotho

  - code: LYD
    name: LibyanDinar
    label: Libyan dinar
    metadata:
      symbol: "ل.د"
      fractions: 1000
      fractional_unit: Dirham
      polity:
        - LBY
      monetary_authority: Central Bank of Libya

  - code: MAD
    name: MoroccanDirham
    label: Moroccan dirham
    metadata:
      symbol: "د.م."
      fractions: 100
      fractional_unit: Centime
      polity:
        - MAR
      monetary_authority: Bank Al-Maghrib

  - code: MDL
    name: MoldovanLeu
    label: Moldovan leu
    metadata:
      symbol: "L"
      fractions: 100
      fractional_unit: Ban
      polity:
        - MDA
      monetary_authority: National Bank of Moldova

  - code: MGA
    name: MalagasyAriary
    label: Malagasy ariary
    metadata:
      symbol: "Ar"
      fractions: 5
      fractional_unit: Iraimbilanja
      polity:
        - MDG
      monetary_authority: Central Bank of Madagascar

  - code: MKD
    name: MacedonianDenar
    label: Macedonian denar
    metadata:
      symbol: "ден"
      fractions: 100
      fractional_unit: Deni
      polity:
        - MKD
      monetary_authority: National Bank of the Republic of Macedonia

  - code: MMK
    name: MyanmarKyat
    label: Myanmar kyat
    metadata:
      symbol: "Ks"
      fractions: 100
      fractional_unit: Pya
      polity:
        - MMR
      monetary_authority: Central Bank of Myanmar

  - code: MNT
    name: MongolianTogrog
    label: Mongolian tögrög
    metadata:
      symbol: "₮"
      fractions: 100
      fractional_unit: Möngö
      polity:
        - MNG
      monetary_authority: Bank of Mongolia

  - code: MOP
    name: MacanesePataca
    label: Macanese pataca
    metadata:
      symbol: "P"
      fractions: 100
      fractional_unit: Avo
      polity:
        - MAC
      monetary_authority: Monetary Authority of Macau

  - code: MRU
    name: MauritanianOuguiya
    label: Mauritanian ouguiya
    metadata:
      symbol: "UM"
      fractions: 5
      fractional_unit: Khoums
      polity:
        - MRT
      monetary_authority: Central Bank of Mauritania

  - code: MUR
    name: MauritianRupee
    label: Mauritian rupee
    metadata:
      symbol: "₨"
      fractions: 100
      fractional_unit: Cent
      polity:
        - MUS
      monetary_authority: Bank of Mauritius

  - code: MVR
    name: MaldivianRufiyaa
    label: Maldivian rufiyaa
    metadata:
      symbol: ".ރ"
      fractions: 100
      fractional_unit: Laari
      polity:
        - MDV
      monetary_authority: Maldives Monetary Authority

  - code: MWK
    name: MalawianKwacha
    label: Malawian kwacha
    metadata:
      symbol: "MK"
      fractions: 100
      fractional_unit: Tambala
      polity:
        - MWI
      monetary_authority: Reserve Bank of Malawi

  - code: MXN
    name: MexicanPeso
    label: Mexican peso
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Centavo
      polity:
        - MEX
      monetary_authority: Bank of Mexico

  - code: MYR
    name: MalaysianRinggit
    label: Malaysian ringgit
    metadata:
      symbol: "RM"
      fractions: 100
      fractional_unit: Sen
      polity:
        - MYS
      monetary_authority: Central Bank of Malaysia

  - code: MZN
    name: MozambicanMetical
    label: Mozambican metical
    metadata:
      symbol: "MT"
      fractions: 100
      fractional_unit: Centavo
      polity:
        - MOZ
      monetary_authority: Bank of Mozambique

  - code: NAD
    name: NamibianDollar
    label: Namibian dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - NAM
      monetary_authority: Bank of Namibia

  - code: NGN
    name: NigerianNaira
    label: Nigerian naira
    metadata:
      symbol: "₦"
      fractions: 100
      fractional_unit: Kobo
      polity:
        - NGA
      monetary_authority: Central Bank of Nigeria

  - code: NIO
    name: NicaraguaCordoba
    label: Nicaraguan córdoba
    metadata:
      symbol: "C$"
      fractions: 100
      fractional_unit: Centavo
      polity:
        - NIC
      monetary_authority: Central Bank of Nicaragua

  - code: NOK
    name: NorwegianKrone
    label: Norwegian krone
    metadata:
      symbol: "kr"
      fractions: 100
      fractional_unit: Øre
      polity:
        - NOR, SJM, BVT
      monetary_authority: Norges Bank

  - code: NPR
    name: NepaleseRupee
    label: Nepalese rupee
    metadata:
      symbol: "₨"
      fractions: 100
      fractional_unit: Paisa
      polity:
        - NPL
      monetary_authority: Nepal Rastra Bank

  - code: NZD
    name: NewZealandDollar
    label: New Zealand dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - NZL
        - COK
        - NIU
        - PCN
        - TKL
      monetary_authority: Reserve Bank of New Zealand

  - code: OMR
    name: OmaniRial
    label: Omani rial
    metadata:
      symbol: "ر.ع."
      fractions: 1000
      fractional_unit: Baisa
      polity:
        - OMN
      monetary_authority: Central Bank of Oman

  - code: PAB
    name: PanamanianBalboa
    label: Panamanian balboa
    metadata:
      symbol: "B/."
      fractions: 100
      fractional_unit: Centésimo
      polity:
        - PAN
      monetary_authority: Central Bank of Oman

  - code: PEN
    name: PeruvianSol
    label: Peruvian sol
    metadata:
      symbol: "S/."
      fractions: 100
      fractional_unit: Céntimo
      polity:
        - PER
      monetary_authority: Central Reserve Bank of Peru

  - code: PGK
    name: PapuaNewGuineanKina
    label: Papua New Guinean kina
    metadata:
      symbol: "K"
      fractions: 100
      fractional_unit: Toea
      polity:
        - PNG
      monetary_authority: Bank of Papua New Guinea

  - code: PHP
    name: PhilippinePeso
    label: Philippine peso
    metadata:
      symbol: "₱"
      fractions: 100
      fractional_unit: Sentimo
      polity:
        - PHL
      monetary_authority: Central Bank of the Philippines

  - code: PKR
    name: PakistaniRupee
    label: Pakistani rupee
    metadata:
      symbol: "₨"
      fractions: 100
      fractional_unit: Paisa
      polity:
        - PAK
      monetary_authority: State Bank of Pakistan

  - code: PLN
    name: PolishZloty
    label: Polish złoty
    metadata:
      symbol: "zł"
      fractions: 100
      fractional_unit: Grosz
      polity:
        - POL
      monetary_authority: National Bank of Poland

  - code: PYG
    name: ParaguayanGuarani
    label: Paraguayan guaraní
    metadata:
      symbol: "₲"
      fractions: 100
      fractional_unit: Céntimo
      polity:
        - PRY
      monetary_authority: Central Bank of Paraguay

  - code: QAR
    name: QatariRiyal
    label: Qatari riyal
    metadata:
      symbol: "ر.ق"
      fractions: 100
      fractional_unit: Dirham
      polity:
        - QAT
      monetary_authority: Qatar Central Bank

  - code: RON
    name: RomanianLeu
    label: Romanian leu
    metadata:
      symbol: "lei"
      fractions: 100
      fractional_unit: Ban
      polity:
        - ROU
      monetary_authority: National Bank of Romania

  - code: RSD
    name: SerbianDinar
    label: Serbian dinar
    metadata:
      symbol: "din"
      fractions: 100
      fractional_unit: Para
      polity:
        - SRB
      monetary_authority: National Bank of Serbia

  - code: RUB
    name: RussianRuble
    label: Russian ruble
    metadata:
      symbol: "₽"
      fractions: 100
      fractional_unit: Kopek
      polity:
        - RUS
      monetary_authority: Bank of Russia

  - code: RWF
    name: RwandanFranc
    label: Rwandan franc
    metadata:
      symbol: "Fr"
      fractions: 100
      fractional_unit: Centime
      polity:
        - RWA
      monetary_authority: National Bank of Rwanda

  - code: SAR
    name: SaudiRiyal
    label: Saudi riyal
    metadata:
      symbol: "ر.س"
      fractions: 100
      fractional_unit: Halala
      polity:
        - SAU
      monetary_authority: Saudi Arabian Monetary Authority

  - code: SBD
    name: SolomonIslandDollar
    label: Solomon Islands dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - SLB
      monetary_authority: Central Bank of Solomon Islands

  - code: SCR
    name: SeychelloisRupee
    label: Seychellois rupee
    metadata:
      symbol: "₨"
      fractions: 100
      fractional_unit: Cent
      polity:
        - SYC
      monetary_authority: Central Bank of Solomon Islands

  - code: SDG
    name: SudanesePound
    label: Sudanese pound
    metadata:
      symbol: "ج.س."
      fractions: 100
      fractional_unit: Piastre
      polity:
        - SSD
      monetary_authority: Bank of Sudan

  - code: SEK
    name: SwedishKrona
    label: Swedish krona
    metadata:
      symbol: "kr"
      fractions: 100
      fractional_unit: Öre
      polity:
        - SWE
      monetary_authority: Sveriges Riksbank

  - code: SGD
    name: SingaporeDollar
    label: Singapore dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - SGP
      monetary_authority: Monetary Authority of Singapore

  - code: SHP
    name: SaintHelenaPound
    label: Saint Helena pound
    metadata:
      symbol: "£"
      fractions: 100
      fractional_unit: Penny
      polity:
        - SHN
      monetary_authority: Government of Saint Helena

  - code: SLL
    name: SierraLeoneanLeone
    label: Sierra Leonean leone
    metadata:
      symbol: "Le"
      fractions: 100
      fractional_unit: Cent
      polity:
        - SLE
      monetary_authority: Bank of Sierra Leone

  - code: SOS
    name: SomaliShilling
    label: Somali shilling
    metadata:
      symbol: "Sl"
      fractions: 100
      fractional_unit: Cent
      polity:
        - SOM
      monetary_authority: Central Bank of Somalia

  - code: SRD
    name: SurilabelseDollar
    label: Surilabelse dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - SUR
      monetary_authority: Central Bank of Surilabel

  - code: SSP
    name: SouthSudanesePound
    label: South Sudanese pound
    metadata:
      symbol: "£"
      fractions: 100
      fractional_unit: Piastre
      polity:
        - SSD
      monetary_authority: Bank of South Sudan

  - code: STN
    name: SaoTomePrincipeDobra
    label: São Tomé and Príncipe dobra
    metadata:
      symbol: "Db"
      fractions: 100
      fractional_unit: Cêntimo
      polity:
        - STP
      monetary_authority: Central Bank of São Tomé and Príncipe

  - code: SYP
    name: SyrianPound
    label: Syrian pound
    metadata:
      symbol: "£"
      fractions: 100
      fractional_unit: Piastre
      polity:
        - SYR
      monetary_authority: Central Bank of Syria

  - code: SZL
    name: SwaziLilangeni
    label: Swazi lilangeni
    metadata:
      symbol: "L"
      fractions: 100
      fractional_unit: Cent
      polity:
        - SWZ
      monetary_authority: Central Bank of Swaziland

  - code: THB
    name: ThaiBaht
    label: Thai baht
    metadata:
      symbol: "฿"
      fractions: 100
      fractional_unit: Satang
      polity:
        - THA
      monetary_authority: Bank of Thailand

  - code: TJS
    name: TajikistaniSomoni
    label: Tajikistani somoni
    metadata:
      symbol: "ЅМ"
      fractions: 100
      fractional_unit: Diram
      polity:
        - TJK
      monetary_authority: National Bank of Tajikistan

  - code: TMT
    name: TurkmenistanManat
    label: Turkmenistan manat
    metadata:
      symbol: "m"
      fractions: 100
      fractional_unit: Tennesi
      polity:
        - TKM
      monetary_authority: Central Bank of Turkmenistan

  - code: TND
    name: TunisianDinar
    label: Tunisian dinar
    metadata:
      symbol: "د.ت"
      fractions: 1000
      fractional_unit: Millime
      polity:
        - TUN
      monetary_authority: Central Bank of Tunisia

  - code: TOP
    name: TonganPaAnga
    label: Tongan paʻanga
    metadata:
      symbol: "T$"
      fractions: 100
      fractional_unit: Seniti
      polity:
        - TON
      monetary_authority: National Reserve Bank of Tonga

  - code: TRY
    name: TurkishLira
    label: Turkish lira
    metadata:
      symbol: "₺"
      fractions: 100
      fractional_unit: Kuruş
      polity:
        - TUR
      monetary_authority: Central Bank of the Republic of Turkey

  - code: TTD
    name: TrinidadTobagoDollar
    label: Trinidad and Tobago dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - TTO
      monetary_authority: Central Bank of Trinidad and Tobago

  - code: TWD
    name: NewTaiwanDollar
    label: New Taiwan dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - TWN
      monetary_authority: Central Bank of the Republic of China (Taiwan)

  - code: TZS
    name: TanzanianShilling
    label: Tanzanian shilling
    metadata:
      symbol: "Sh"
      fractions: 100
      fractional_unit: Cent
      polity:
        - TZA
      monetary_authority: Bank of Tanzania

  - code: UAH
    name: UkrainianHryvnia
    label: Ukrainian hryvnia
    metadata:
      symbol: "₴"
      fractions: 100
      fractional_unit: Kopiyka
      polity:
        - UKR
      monetary_authority: National Bank of Ukraine

  - code: UGX
    name: UgandanShilling
    label: Ugandan shilling
    metadata:
      symbol: "Sh"
      fractions: 100
      fractional_unit: Cent
      polity:
        - UGA
      monetary_authority: Bank of Uganda

  - code: USD
    name: UnitedStatesDollar
    label: United States dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - USA
        - ASM
        - VGB
        - BES
        - ECU
        - SLV
        - GUM
        - HTI
        - MHL
        - FSM
        - MNP
        - PLW
        - PAN
        - PRI
        - TLS
        - TCA
        - VIR
        - UMI
      monetary_authority: Federal Reserve

  - code: UYU
    name: UruguayanPeso
    label: Uruguayan peso
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Centésimo
      polity:
        - URY
      monetary_authority: Central Bank of Uruguay

  - code: UYU
    name: UruguayanPeso
    label: Uruguayan peso
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Centésimo
      polity:
        - URY
      monetary_authority: Central Bank of Uruguay

  - code: UZS
    name: UzbekistaniSom
    label: Uzbekistani soʻm
    metadata:
      symbol: "so'm"
      fractions: 100
      fractional_unit: Tiyin
      polity:
        - UZB
      monetary_authority: Central Bank of the Republic of Uzbekistan

  - code: VES
    name: VenezuelanBolivarSoberano
    label: Venezuelan bolívar soberano
    metadata:
      symbol: "Bs."
      fractions: 100
      fractional_unit: Céntimo
      polity:
        - VEN
      monetary_authority: Central Bank of Venezuela

  - code: VND
    name: VietlabelseDong
    label: Vietlabelse đồng
    metadata:
      symbol: "₫"
      fractions: 100
      fractional_unit: Hào
      polity:
        - VNM
      monetary_authority: State Bank of Vietnam

  - code: VUV
    name: VanuatuVatu
    label: Vanuatu vatu
    metadata:
      symbol: "Vt"
      fractions: 1
      fractional_unit: ~
      polity:
        - VUT
      monetary_authority: Reserve Bank of Vanuatu

  - code: WST
    name: SamoanTala
    label: Samoan tālā
    metadata:
      symbol: "T"
      fractions: 100
      fractional_unit: Sene
      polity:
        - WSM
      monetary_authority: Central Bank of Samoa

  - code: XAF
    name: CentralAfricanCFAFranc
    label: Central African CFA franc
    metadata:
      symbol: "T"
      fractions: 100
      fractional_unit: Sene
      polity:
        - CMR
        - CAF
        - COG
        - TCD
        - GNQ
        - GAB
      monetary_authority: Central Bank of Samoa

  - code: XCD
    name: EastCaribbeanDollar
    label: East Caribbean dollar
    metadata:
      symbol: "$"
      fractions: 100
      fractional_unit: Cent
      polity:
        - AIA
        - ATG
        - DMA
        - GRD
        - MSR
        - KNA
        - LCA
        - VCT
      monetary_authority: Eastern Caribbean Central Bank

  - code: XOF
    name: WestAfricanCFAFranc
    label: West African CFA franc
    metadata:
      symbol: "Fr"
      fractions: 100
      fractional_unit: Centime
      polity:
        - BEN
        - BFA
        - CIV
        - GNB
        - MLI
        - NER
        - SEN
        - TGO
      monetary_authority: Central Bank of West African States

  - code: YER
    name: YemeniRial
    label: Yemeni rial
    metadata:
      symbol: "﷼"
      fractions: 100
      fractional_unit: Fils
      polity:
        - YEM
      monetary_authority: Central Bank of Yemen

  - code: ZAR
    name: SouthAfricanRand
    label: South African rand
    metadata:
      symbol: "R"
      fractions: 100
      fractional_unit: Cent
      polity:
        - ZAF
      monetary_authority: South African Reserve Bank

  - code: ZMW
    name: ZambianKwacha
    label: Zambian kwacha
    metadata:
      symbol: "ZK"
      fractions: 100
      fractional_unit: Ngwee
      polity:
        - ZMB
      monetary_authority: Bank of Zambia

`

// Entities - Legal Entities & Ownership Structures. 1 character.
var yamlEntities = `

    - code: I
      name: Individual # (Natural Person)
      label: "Individual"
      metadata:
        type: Legal
        roles:
          owners:
            principal: null
          administrators:
            legalGuardian: []
            agent: [] # (granted by the principal)
          general:
            accountant: []
            advisor: []
            lawyer: []
            manager: []
            trader: []

    - code: P
      name: PublicCompany
      label: "Public Company Limited by Shares"
      metadata:
        type: Legal
        roles:
          owners:
            shareholder: []
            significantShareholder: []
          administrators: # Board of Directors
            chairman: null
            director: []
          managers:
            ceo: null
            coo: null
            cfo: null
            cto: null
            executive: []
            secretary: null
          general:
            accountant: []
            advisor: []
            employee: []
            lawyer: []
            manager: []
            trader: []

    - code: C
      name: PrivateCompany
      label: "Private Company Limited by Shares"
      metadata:
        type: Legal
        roles:
          owners:
            shareholder: []
            significantShareholder: []
          administrators:
            chairman: null
            director: []
            secretary: null
          managers:
            ceo: null
            coo: null
            cfo: null
            cto: null
            executive: []
          general:
            accountant: []
            advisor: []
            employee: []
            lawyer: []
            manager: []
            trader: []

    - code: L
      name: LimitedPartnership
      label: "Limited Partnership"
      metadata:
        type: Legal
        roles:
          owners:
            partner: []
          managers:
            managingPartner: []
          general:
            accountant: []
            advisor: []
            employee: []
            lawyer: []
            manager: []
            trader: []

    - code: U
      name: UnlimitedPartnership
      label: "Unlimited Partnership" # (General Partnership, Marriage, Civil Union, Common Law Marriage, Domestic Partnership,  )
      metadata:
        type: Legal
        roles:
          owners:
            partner: []
          managers:
            managingPartner: []
          general:
            accountant: []
            advisor: []
            employee: []
            lawyer: []
            manager: []
            trader: []

    - code: T
      name: SoleProprietorship
      label: "Sole Proprietorship"
      metadata:
        type: Legal
        roles:
          owners:
            proprietor: null
          administrators:
            agent: [] # (granted by the proprietor)
          general:
            accountant: []
            advisor: []
            employee: []
            lawyer: []
            manager: []
            trader: []

    - code: S
      name: StatutoryCompany
      label: "Statutory Company"
      metadata:
        type: Legal
        roles:
          administrators:
            chairman: null
            director: []
            secretary: null
          managers:
            ceo: null
            coo: null
            cfo: null
            cto: null
            executive: []
            secretary: null
          general:
            accountant: []
            advisor: []
            employee: []
            lawyer: []
            manager: []
            trader: []

    - code: O
      name: NonProfitOrganization
      label: "Non-Profit Organization"
      metadata:
        type: Legal
        role:
          administrators:
            chairman: null
            director: []
            secretary: null
          managers:
            ceo: null
            coo: null
            cfo: null
            cto: null
            executive: []
          general:
            accountant: []
            advisor: []
            employee: []
            lawyer: []
            manager: []
            trader: []

    - code: N
      name: NationState
      label: "Nation State" # (Sovereign State)
      metadata:
        type: Legal

    - code: G
      name: GovernmentAgency
      label: "Government Agency"
      metadata:
        type: Legal

    - code: U
      name: UnitTrust
      label: "Unit Trust"
      metadata:
        type: Ownership
        roles:
          owners:
            unitholder: []
          administrators:
            protector: []
            trustee: []
          general:
            accountant: []
            advisor: []
            custodian: []
            employee: []
            lawyer: []
            manager: []
            settlor: []
            trader: []

    - code: D
      name: DiscretionaryTrust
      label: "Discretionary Trust" # (Family Trust)
      metadata:
        type: Ownership
        roles:
          owners:
            beneficiary: []
          administrators:
            protector: []
            trustee: []
          general:
            accountant: []
            advisor: []
            custodian: []
            employee: []
            lawyer: []
            manager: []
            settlor: []
            trader: []

`

// Polities - Polities (eg. Countries/Nation-States (ISO-3166 Alpha-3),
// Political Unions, International Organizations, etc.). Based on a 3
// character code.
var yamlPolities = `
  - code: ALA
    name: Aaland Islands
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/5/52/Flag_of_%C3%85land.svg"

  - code: AFG
    name: Afghanistan
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/9/9a/Flag_of_Afghanistan.svg"

  - code: ALB
    name: Albania
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/3/36/Flag_of_Albania.svg"

  - code: DZA
    name: Algeria
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/7/77/Flag_of_Algeria.svg"

  - code: ASM
    name: American Samoa
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/8/87/Flag_of_American_Samoa.svg"

  - code: AND
    name: Andorra
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/1/19/Flag_of_Andorra.svg"

  - code: AGO
    name: Angola
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/9/9d/Flag_of_Angola.svg"

  - code: AIA
    name: Anguilla
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/b/b4/Flag_of_Anguilla.svg"

  - code: ATA
    name: Antarctica
    metadata:
      states: ~
      flag:

  - code: ATG
    name: Antigua and Barbuda
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/8/89/Flag_of_Antigua_and_Barbuda.svg"

  - code: ARG
    name: Argentina
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/1/1a/Flag_of_Argentina.svg"

  - code: ARM
    name: Armenia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/2/2f/Flag_of_Armenia.svg"

  - code: ABW
    name: Aruba
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/f/f6/Flag_of_Aruba.svg"

  - code: AU
    name: African Union
    metadata:
      states:
        DZA: ~
        AGO: ~
        BEN: ~
        BWA: ~
        BFA: ~
        BDI: ~
        CPV: ~
        CMR: ~
        CAF: ~
        TCD: ~
        COM: ~
        COD: ~
        DJI: ~
        EGY: ~
        GNQ: ~
        ERI: ~
        SWZ: ~
        ETH: ~
        GAB: ~
        GMB: ~
        GHA: ~
        GIN: ~
        GNB: ~
        CIV: ~
        KEN: ~
        LSO: ~
        LBR: ~
        LBY: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/5/51/Flag_of_the_African_Union.svg"

  - code: AUS
    name: Australia
    metadata:
      states:
        AUNSW: New South Wales
        AUQLD: Queensland
        AUSA: South Australia
        AUTAS: Tasmania
        AUVIC: Victoria
        AUWA: Western Australia
        AUACT: Australian Capital Territory
        AUNT: Northern Territory
        AUJBT: Jervis Bay Territory
        AUCX: Christmas Island
        AUCC: Cocos (Keening) Island
        AUHM: Heard Island and McDonalds Islands
        AUNF: Norfolk Island
      flag: "https://upload.wikimedia.org/wikipedia/commons/b/b9/Flag_of_Australia.svg"
      fiscal_year: 0701-0630
      gov_fiscal_year: 0701-0630

  - code: AUT
    name: Austria
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/4/41/Flag_of_Austria.svg"
      fiscal_year: 0101-1231
      gov_fiscal_year: 0101-1231

  - code: AZE
    name: Azerbaijan
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/d/dd/Flag_of_Azerbaijan.svg"

  - code: BHS
    name: The Bahamas
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/9/93/Flag_of_the_Bahamas.svg"

  - code: BHR
    name: Bahrain
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/2/2c/Flag_of_Bahrain.svg"

  - code: BGD
    name: Bangladesh
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/f/f9/Flag_of_Bangladesh.svg"
      fiscal_year: 0701-0630
      gov_fiscal_year: 0701-0630

  - code: BRB
    name: Barbados
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/e/ef/Flag_of_Barbados.svg"

  - code: BLR
    name: Belarus
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/8/85/Flag_of_Belarus.svg"

  - code: BEL
    name: Belgium
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/6/65/Flag_of_Belgium.svg"
      fiscal_year: 0101-1231
      gov_fiscal_year: 0101-1231

  - code: BLZ
    name: Belize
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/e/e7/Flag_of_Belize.svg"

  - code: BEN
    name: Benin
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/0a/Flag_of_Benin.svg"

  - code: BMU
    name: Bermuda
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/b/bf/Flag_of_Bermuda.svg"

  - code: BTN
    name: Bhutan
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/9/91/Flag_of_Bhutan.svg"

  - code: BOL
    name: Bolivia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/d/de/Flag_of_Bolivia_%28state%29.svg"

  - code: BES
    name: Bonaire, St Eustasuis and Saba
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/2/20/Flag_of_the_Netherlands.svg"

  - code: BIH
    name: Bosnia and Herzegovina
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/b/bf/Flag_of_Bosnia_and_Herzegovina.svg"

  - code: BWA
    name: Botswana
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/f/fa/Flag_of_Botswana.svg"

  - code: BVT
    name: Bouvet Island
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/d/d9/Flag_of_Norway.svg"

  - code: BRA
    name: Brazil
    metadata:
      states:
        BRAC: Acre
        BRAL: Alagoas
        BRAP: Amapá
        BRAM: Amazonas
        BRBA: Bahia
        BRCE: Ceará
        BRES: Espírito Santo
        BRDF: Federal District
        BRGO: Goiás
        BRMA: Maranhão
        BRMT: Mato Grosso
        BRMS: Mato Grosso do Sul
        BRMG: Minas Gerais
        BRPA: Pará
        BRPB: Paraíba
        BRPR: Paraná
        BRPE: Pernambuco
        BRPI: Piauí
        BRRJ: Rio de Janeiro
        BRRN: Rio Grande do Norte
        BRRS: Rio Grande do Sul
        BRRO: Rondônia
        BRRR: Roraima
        BRSC: Santa Catarina
        BRSP: São Paulo
        BRSE: Sergipe
        BRTO: Tocantins
      flag: "https://upload.wikimedia.org/wikipedia/en/0/05/Flag_of_Brazil.svg"
      fiscal_year: 0101-1231
      gov_fiscal_year: 0101-1231

  - code: IOT
    name: British Indian Ocean Territory
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/6/6e/Flag_of_the_British_Indian_Ocean_Territory.svg"

  - code: VGB
    name: British Virgin Islands
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/4/42/Flag_of_the_British_Virgin_Islands.svg"

  - code: BRN
    name: Brunei
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/9/9c/Flag_of_Brunei.svg"

  - code: BGR
    name: Bulgaria
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/9/9a/Flag_of_Bulgaria.svg"

  - code: BFA
    name: Burkina Faso
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/3/31/Flag_of_Burkina_Faso.svg"

  - code: BDI
    name: Burundi
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/5/50/Flag_of_Burundi.svg"

  - code: KHM
    name: Cambodia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/8/83/Flag_of_Cambodia.svg"

  - code: CMR
    name: Cameroon
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/4/4f/Flag_of_Cameroon.svg"

  - code: CAN
    name: Canada
    metadata:
      states:
        CAAB: Alberta
        CABC: British Columbia
        CAMB: Manitoba
        CANB: New Brunswick
        CANL: Newfoundland and Labrador
        CANS: Nova Scotia
        CAON: Ontario
        CAPE: Prince Edward Island
        CAQC: Quebec
        CASK: Saskatchewan
        CANT: Northwest Territories
        CANU: Nunavut
        CAYT: Yukon
      flag: "https://upload.wikimedia.org/wikipedia/commons/d/d9/Flag_of_Canada_%28Pantone%29.svg"
      fiscal_year: 0101-1231
      gov_fiscal_year: 0401-0331

  - code: CPV
    name: Cape Verde
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/3/38/Flag_of_Cape_Verde.svg"

  - code: CYM
    name: Cayman Islands
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/0f/Flag_of_the_Cayman_Islands.svg"

  - code: CAF
    name: Central African Republic
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/6/6f/Flag_of_the_Central_African_Republic.svg"

  - code: TCD
    name: Chad
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/4/4b/Flag_of_Chad.svg"

  - code: CHL
    name: Chile
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/7/78/Flag_of_Chile.svg"

  - code: CHN
    name: China
    metadata:
      states:
        CNBJ: Beijing
        CNTJ: Tianjin
        CNHE: Hebei
        CNSX: Shanxi
        CNNM: Nei Mongol (mn)
        CNLN: Liaoning
        CNJL: Jilin
        CNHL: Heilongjiang
        CNSH: Shanghai
        CNJS: Jiangsu
        CNZJ: Zhejiang
        CNAH: Anhui
        CNFJ: Fujian
        CNJX: Jiangxi
        CNSD: Shandong
        CNHA: Henan
        CNHB: Hubei
        CNHN: Hunan
        CNGD: Guangdong
        CNGX: Guangxi
        CNHI: Hainan
        CNCQ: Chongqing
        CNSC: Sichuan
        CNGZ: Guizhou
        CNYN: Yunnan
        CNXZ: Xizang
        CNSN: Shaanxi
        CNGS: Gansu
        CNQH: Qinghai
        CNNX: Ningxia
        CNXJ: Xinjiang
        CNTW: Taiwan
        CNHK: Hong Kong (Xianggang)
        CNMC: Macao (Aomen)
      flag: "https://upload.wikimedia.org/wikipedia/commons/f/fa/Flag_of_the_People%27s_Republic_of_China.svg"
      fiscal_year: 0101-1231
      gov_fiscal_year: 0101-1231

  - code: CXR
    name: Christmas Island
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/6/67/Flag_of_Christmas_Island.svg"

  - code: CCK
    name: Cocos Islands
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/7/74/Flag_of_the_Cocos_%28Keeling%29_Islands.svg"

  - code: COL
    name: Colombia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/2/21/Flag_of_Colombia.svg"

  - code: COM
    name: Comoros
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/9/94/Flag_of_the_Comoros.svg"

  - code: COG
    name: Congo-Brazzaville
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/9/92/Flag_of_the_Republic_of_the_Congo.svg"

  - code: COD
    name: Congo-Kinshasa
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/6/6f/Flag_of_the_Democratic_Republic_of_the_Congo.svg"

  - code: COK
    name: Cook Islands
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/3/35/Flag_of_the_Cook_Islands.svg"

  - code: CRI
    name: Costa Rica
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/b/bc/Flag_of_Costa_Rica_%28state%29.svg"
      fiscal_year: 1001-0931
      gov_fiscal_year: 1001-0931

  - code: HRV
    name: Croatia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/1/1b/Flag_of_Croatia.svg"

  - code: CUB
    name: Cuba
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/b/bd/Flag_of_Cuba.svg"

  - code: CUW
    name: Curacao
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/b/b1/Flag_of_Cura%C3%A7ao.svg"

  - code: CYP
    name: Cyprus
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/d/d4/Flag_of_Cyprus.svg"

  - code: CZE
    name: Czech Republic
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/c/cb/Flag_of_the_Czech_Republic.svg"

  - code: DNK
    name: Denmark
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/9/9c/Flag_of_Denmark.svg"

  - code: DJI
    name: Djibouti
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/3/34/Flag_of_Djibouti.svg"

  - code: DMA
    name: Dominica
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/c/c4/Flag_of_Dominica.svg"

  - code: DOM
    name: Dominican Republic
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/9/9f/Flag_of_the_Dominican_Republic.svg"

  - code: TLS
    name: East Timor
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/2/26/Flag_of_East_Timor.svg"

  - code: ECU
    name: Ecuador
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/e/e8/Flag_of_Ecuador.svg"

  - code: EGY
    name: Egypt
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/f/fe/Flag_of_Egypt.svg"
      fiscal_year: 0701-0630
      gov_fiscal_year: 0701-0630

  - code: SLV
    name: El Salvador
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/3/34/Flag_of_El_Salvador.svg"

  - code: GNQ
    name: Equatorial Guinea
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/3/31/Flag_of_Equatorial_Guinea.svg"

  - code: ERI
    name: Eritrea
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/2/29/Flag_of_Eritrea.svg"

  - code: EST
    name: Estonia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/8/8f/Flag_of_Estonia.svg"

  - code: ETH
    name: Ethiopia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/7/71/Flag_of_Ethiopia.svg"
      fiscal_year: 0708-0707
      gov_fiscal_year: 0708-0707

  - code: EU
    name: European Union
    metadata:
      states:
        AUT: ~
        BEL: ~
        BGR: ~
        HRV: ~
        CYP: ~
        CZE: ~
        DNK: ~
        EST: ~
        FIN: ~
        FRA: ~
        DEU: ~
        GRC: ~
        HUN: ~
        IRL: ~
        ITA: ~
        LVA: ~
        LTU: ~
        LUX: ~
        MLT: ~
        NLD: ~
        POL: ~
        PRT: ~
        ROU: ~
        SVK: ~
        SVN: ~
        ESP: ~
        SWE: ~
        GBR: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/b/b7/Flag_of_Europe.svg"

  - code: FLK
    name: Falkland Islands
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/8/83/Flag_of_the_Falkland_Islands.svg"

  - code: FRO
    name: Faroe Islands
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/3/3c/Flag_of_the_Faroe_Islands.svg"

  - code: FJI
    name: Fiji
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/b/ba/Flag_of_Fiji.svg"

  - code: FIN
    name: Finland
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/b/bc/Flag_of_Finland.svg"

  - code: FRA
    name: France
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/en/c/c3/Flag_of_France.svg"

  - code: GUF
    name: French Guiana
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/en/c/c3/Flag_of_France.svg"

  - code: PYF
    name: French Polynesia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/d/db/Flag_of_French_Polynesia.svg"

  - code: ATF
    name: French Southern and Antarctic Lands
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/a/a7/Flag_of_the_French_Southern_and_Antarctic_Lands.svg"

  - code: GAB
    name: Gabon
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/04/Flag_of_Gabon.svg"

  - code: GMB
    name: The Gambia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/7/77/Flag_of_The_Gambia.svg"

  - code: GEO
    name: Georgia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/0f/Flag_of_Georgia.svg"

  - code: DEU
    name: Germany
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/en/b/ba/Flag_of_Germany.svg"
      fiscal_year: 0101-1231
      gov_fiscal_year: 0101-1231

  - code: GHA
    name: Ghana
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/1/19/Flag_of_Ghana.svg"

  - code: GIB
    name: Gibraltar
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/02/Flag_of_Gibraltar.svg"

  - code: GRC
    name: Greece
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/5/5c/Flag_of_Greece.svg"
      fiscal_year: 0101-1231
      gov_fiscal_year: 0101-1231

  - code: GRL
    name: Greenland
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/09/Flag_of_Greenland.svg"

  - code: GRD
    name: Grenada
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/b/bc/Flag_of_Grenada.svg"

  - code: GLP
    name: Guadeloupe
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/en/c/c3/Flag_of_France.svg"

  - code: GUM
    name: Guam
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/07/Flag_of_Guam.svg"

  - code: GTM
    name: Guatemala
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/e/ec/Flag_of_Guatemala.svg"

  - code: GGY
    name: Guernsey
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/f/fa/Flag_of_Guernsey.svg"

  - code: GIN
    name: Guinea
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/e/ed/Flag_of_Guinea.svg"

  - code: GNB
    name: Guinea-Bissau
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/01/Flag_of_Guinea-Bissau.svg"

  - code: GUY
    name: Guyana
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/9/99/Flag_of_Guyana.svg"

  - code: HTI
    name: Haiti
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/5/56/Flag_of_Haiti.svg"

  - code: HMD
    name: Heard Island and McDonald Islands
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/8/88/Flag_of_Australia_%28converted%29.svg"

  - code: HND
    name: Honduras
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/8/82/Flag_of_Honduras.svg"

  - code: HKG
    name: Hong Kong
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/5/5b/Flag_of_Hong_Kong.svg"
      fiscal_year: 0401-0331
      gov_fiscal_year: 0401-0331

  - code: HUN
    name: Hungary
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/c/c1/Flag_of_Hungary.svg"

  - code: ISL
    name: Iceland
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/c/ce/Flag_of_Iceland.svg"

  - code: IND
    name: India
    metadata:
      states:
        INAN: Andaman and Nicobar Islands
        INAP: Andhra Pradesh
        INAR: Arunachal Pradesh
        INAS: Assam
        INBR: Bihar
        INCH: Chandigarh
        INCT: Chhattisgarh
        INDN: Dadra and Nagar Haveli
        INDD: Daman and Diu
        INDL: Delhi
        INGA: Goa
        INGJ: Gujarat
        INHR: Haryana
        INHP: Himachal Pradesh
        INJK: Jammu and Kashmir
        INJH: Jharkhand
        INKA: Karnataka
        INKL: Kerala
        INLD: Lakshadweep
        INMP: Madhya Pradesh
        INMH: Maharashtra
        INMN: Manipur
        INML: Meghalaya
        INMZ: Mizoram
        INNL: Nagaland
        INOR: Odisha (formerly known as Orissa)
        INPY: Puducherry (Pondicherry)
        INPB: Punjab
        INRJ: Rajasthan
        INSK: Sikkim
        INTN: Tamil Nadu
        INTG: Telangana
        INTR: Tripura
        INUT: Uttarakhand
        INUP: Uttar Pradesh
        INWB: West Bengal
      flag: "https://upload.wikimedia.org/wikipedia/en/4/41/Flag_of_India.svg"
      fiscal_year: 0401-0331
      gov_fiscal_year: 0401-0331

  - code: IDN
    name: Indonesia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/9/9f/Flag_of_Indonesia.svg"
      fiscal_year: 0101-1231
      gov_fiscal_year: 0101-1231

  - code: IRN
    name: Iran
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/c/ca/Flag_of_Iran.svg"

  - code: IRQ
    name: Iraq
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/f/f6/Flag_of_Iraq.svg"

  - code: IRL
    name: Ireland
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/4/45/Flag_of_Ireland.svg"
      fiscal_year: 0101-1231
      gov_fiscal_year: 0101-1231

  - code: IMN
    name: Isle of Man
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/5/5d/Flag_of_the_Isle_of_Mann.svg"

  - code: ISR
    name: Israel
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/d/d4/Flag_of_Israel.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231

  - code: ITA
    name: Italy
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/en/0/03/Flag_of_Italy.svg"

  - code: CIV
    name: Ivory Coast
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/f/fe/Flag_of_C%C3%B4te_d%27Ivoire.svg"

  - code: JAM
    name: Jamaica
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/0a/Flag_of_Jamaica.svg"

  - code: JPN
    name: Japan
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/en/9/9e/Flag_of_Japan.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0401-0331

  - code: JEY
    name: Jersey
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/1/1c/Flag_of_Jersey.svg"

  - code: JOR
    name: Jordan
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/c/c0/Flag_of_Jordan.svg"

  - code: KAZ
    name: Kazakhstan
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/d/d3/Flag_of_Kazakhstan.svg"

  - code: KEN
    name: Kenya
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/4/49/Flag_of_Kenya.svg"

  - code: KIR
    name: Kiribati
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/d/d3/Flag_of_Kiribati.svg"

  - code: KWT
    name: Kuwait
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/a/aa/Flag_of_Kuwait.svg"

  - code: KGZ
    name: Kyrgyzstan
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/c/c7/Flag_of_Kyrgyzstan.svg"

  - code: LAO
    name: Laos
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/5/56/Flag_of_Laos.svg"

  - code: LVA
    name: Latvia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/8/84/Flag_of_Latvia.svg"

  - code: LBN
    name: Lebanon
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/5/59/Flag_of_Lebanon.svg"

  - code: LSO
    name: Lesotho
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/4/4a/Flag_of_Lesotho.svg"

  - code: LBR
    name: Liberia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/b/b8/Flag_of_Liberia.svg"

  - code: LBY
    name: Libya
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/05/Flag_of_Libya.svg"

  - code: LIE
    name: Liechtenstein
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/4/47/Flag_of_Liechtenstein.svg"

  - code: LTU
    name: Lithuania
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/1/11/Flag_of_Lithuania.svg"

  - code: LUX
    name: Luxembourg
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/d/da/Flag_of_Luxembourg.svg"

  - code: MAC
    name: Macau
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/6/63/Flag_of_Macau.svg"

  - code: MKD
    name: Macedonia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/f/f8/Flag_of_Macedonia.svg"

  - code: MDG
    name: Madagascar
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/b/bc/Flag_of_Madagascar.svg"

  - code: MWI
    name: Malawi
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/d/d1/Flag_of_Malawi.svg"

  - code: MYS
    name: Malaysia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/6/66/Flag_of_Malaysia.svg"

  - code: MDV
    name: Maldives
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/0f/Flag_of_Maldives.svg"

  - code: MLI
    name: Mali
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/9/92/Flag_of_Mali.svg"

  - code: MLT
    name: Malta
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/7/73/Flag_of_Malta.svg"

  - code: MHL
    name: Marshall Islands
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/2/2e/Flag_of_the_Marshall_Islands.svg"

  - code: MTQ
    name: Martinique
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/en/c/c3/Flag_of_France.svg"

  - code: MRT
    name: Mauritania
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/3/30/New_National_Flag_of_Mauritania.png"

  - code: MUS
    name: Mauritius
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/7/77/Flag_of_Mauritius.svg"

  - code: MYT
    name: Mayotte
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/4/4a/Flag_of_Mayotte_%28local%29.svg"

  - code: MEX
    name: Mexico
    metadata:
      states:
      MXAGU: Aguascalientes
      MXBCN: Baja California
      MXBCS: Baja California Sur
      MXCHP: Chiapas
      MXCHH: Chihuahua
      MXCAM: Campeche
      MXCOA: Coahuila
      MXCOL: Colima
      MXDIF: Distrito Federal
      MXDUR: Durango
      MXGUA: Guanajuato
      MXGRO: Guerrero
      MXHID: Hidalgo
      MXJAL: Jalisco
      MXMEX: Mexico (Federal District)
      MXMIC: Michoacán
      MXMOR: Morelos
      MXNAY: Nayarit
      MXNLE: Nuevo León
      MXOAX: Oaxaca
      MXPUE: Puebla
      MXQUE: Querétaro
      MXROO: Quintana Roo
      MXSLP: San Luis Potosí
      MXSIN: Sinaloa
      MXSON: Sonora
      MXTAB: Tabasco
      MXTAM: Tamaulipas
      MXTLA: Tlaxcala
      MXVER: Veracruz
      MXYUC: Yucatán
      MXZAC: Zacatecas
      flag: "https://upload.wikimedia.org/wikipedia/commons/f/fc/Flag_of_Mexico.svg"

  - code: FSM
    name: Micronesia
    metadata:
      states: ~
      flag:

  - code: MDA
    name: Moldova
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/2/27/Flag_of_Moldova.svg"

  - code: MCO
    name: Monaco
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/e/ea/Flag_of_Monaco.svg"

  - code: MNG
    name: Mongolia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/4/4c/Flag_of_Mongolia.svg"

  - code: MNE
    name: Montenegro
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/6/64/Flag_of_Montenegro.svg"

  - code: MSR
    name: Montserrat
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/d/d0/Flag_of_Montserrat.svg"

  - code: MAR
    name: Morocco
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/2/2c/Flag_of_Morocco.svg"

  - code: MOZ
    name: Mozambique
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/d/d0/Flag_of_Mozambique.svg"

  - code: MMR
    name: Myanmar
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/8/8c/Flag_of_Myanmar.svg"

  - code: NAM
    name: Namibia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/00/Flag_of_Namibia.svg"

  - code: NRU
    name: Nauru
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/3/30/Flag_of_Nauru.svg"

  - code: NPL
    name: Nepal
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/9/9b/Flag_of_Nepal.svg"
      fiscal_year: 0101-1231
      gov_fiscal_year: 0101-1231

  - code: NLD
    name: Netherlands
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/2/20/Flag_of_the_Netherlands.svg"
      fiscal_year: 0101-1231
      gov_fiscal_year: 0101-1231

  - code: NCL
    name: New Caledonia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/en/c/c3/Flag_of_France.svg"

  - code: NZL
    name: New Zealand
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/3/3e/Flag_of_New_Zealand.svg"

  - code: NIC
    name: Nicaragua
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/1/19/Flag_of_Nicaragua.svg"

  - code: NER
    name: Niger
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/f/f4/Flag_of_Niger.svg"

  - code: NGA
    name: Nigeria
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/7/79/Flag_of_Nigeria.svg"

  - code: NIU
    name: Niue
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/01/Flag_of_Niue.svg"

  - code: NFK
    name: Norfolk and Philip Island
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/4/48/Flag_of_Norfolk_Island.svg"

  - code: PRK
    name: North Korea
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/5/51/Flag_of_North_Korea.svg"

  - code: MNP
    name: Northern Mariana Islands
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/e/e0/Flag_of_the_Northern_Mariana_Islands.svg"

  - code: NOR
    name: Norway
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/d/d9/Flag_of_Norway.svg"

  - code: OMN
    name: Oman
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/d/dd/Flag_of_Oman.svg"

  - code: PAK
    name: Pakistan
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/3/32/Flag_of_Pakistan.svg"

  - code: PLW
    name: Palau
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/4/48/Flag_of_Palau.svg"

  - code: PSE
    name: Palestinian Territory
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/00/Flag_of_Palestine.svg"

  - code: PAN
    name: Panama
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/a/ab/Flag_of_Panama.svg"

  - code: PNG
    name: Papua New Guinea
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/e/e3/Flag_of_Papua_New_Guinea.svg"

  - code: PRY
    name: Paraguay
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/2/27/Flag_of_Paraguay.svg"

  - code: PER
    name: Peru
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/d/df/Flag_of_Peru_%28state%29.svg"

  - code: PHL
    name: Philippines
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/9/99/Flag_of_the_Philippines.svg"

  - code: PCN
    name: Pitcairn Islands
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/8/88/Flag_of_the_Pitcairn_Islands.svg"

  - code: POL
    name: Poland
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/en/1/12/Flag_of_Poland.svg"

  - code: PRT
    name: Portugal
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/5/5c/Flag_of_Portugal.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231

  - code: PRI
    name: Puerto Rico
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/2/28/Flag_of_Puerto_Rico.svg"

  - code: QAT
    name: Qatar
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/6/65/Flag_of_Qatar.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231

  - code: REU
    name: Réunion
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/5/5a/Flag_of_R%C3%A9union.svg"

  - code: ROU
    name: Romania
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/7/73/Flag_of_Romania.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231

  - code: RUS
    name: Russia
    metadata:
      states:
        RUAD: Adygeya, Respublika
        RUAL: Altay, Respublika
        RUBA: Bashkortostan, Respublika
        RUBU: Buryatiya, Respublika
        RUCE: Chechenskaya Respublika
        RUCU: Chuvashskaya Respublika
        RUDA: Dagestan, Respublika
        RUIN: Ingushetiya, Respublika
        RUKB: Kabardino-Balkarskaya Respublika
        RUKL: Kalmykiya, Respublika
        RUKC: Karachayevo-Cherkesskaya Respubl.
        RUKR: Kareliya, Respublika
        RUKK: Khakasiya, Respublika
        RUKO: Komi, Respublika
        RUME: Mariy El, Respublika
        RUMO: Mordoviya, Respublika
        RUSA: Sakha, Respublika
        RUSE: Severnaya Osetiya-Alaniya, Respubl.
        RUTA: Tatarstan, Respublika
        RUTY: Tyva, Respublika
        RUUD: Udmurtskaya Respublika
        RUALT: Altayskiy kray
        RUKAM: Kamchatskiy kray
        RUKHA: Khabarovskiy kray
        RUKDA: Krasnodarskiy kray
        RUKYA: Krasnoyarskiy kray
        RUPER: Permskiy kray
        RUPRI: Primorskiy kray
        RUSTA: Stavropol"skiy kray
        RUZAB: Zabaykal"skiy kray
        RUAMU: Amurskaya oblast"
        RUARK: Arkhangel"skaya oblast"
        RUAST: Astrakhanskaya oblast"
        RUBEL: Belgorodskaya oblast"
        RUBRY: Bryanskaya oblast"
        RUCHE: Chelyabinskaya oblast"
        RUIRK: Irkutskaya oblast"
        RUIVA: Ivanovskaya oblast"
        RUKGD: Kaliningradskaya oblast"
        RUKLU: Kaluzhskaya oblast"
        RUKEM: Kemerovskaya oblast"
        RUKIR: Kirovskaya oblast"
        RUKOS: Kostromskaya oblast"
        RUKGN: Kurganskaya oblast"
        RUKRS: Kurskaya oblast"
        RULEN: Leningradskaya oblast"
        RULIP: Lipetskaya oblast"
        RUMAG: Magadanskaya oblast"
        RUMOS: Moskovskaya oblast"
        RUMUR: Murmanskaya oblast"
        RUNIZ: Nizhegorodskaya oblast"
        RUNGR: Novgorodskaya oblast"
        RUNVS: Novosibirskaya oblast"
        RUOMS: Omskaya oblast"
        RUORE: Orenburgskaya oblast"
        RUORL: Orlovskaya oblast"
        RUPNZ: Penzenskaya oblast"
        RUPSK: Pskovskaya oblast"
        RUROS: Rostovskaya oblast"
        RURYA: Ryazanskaya oblast"
        RUSAK: Sakhalinskaya oblast"
        RUSAM: Samarskaya oblast"
        RUSAR: Saratovskaya oblast"
        RUSMO: Smolenskaya oblast"
        RUSVE: Sverdlovskaya oblast"
        RUTAM: Tambovskaya oblast"
        RUTOM: Tomskaya oblast"
        RUTUL: Tul"skaya oblast"
        RUTVE: Tverskaya oblast"
        RUTYU: Tyumenskaya oblast"
        RUULY: Ul"yanovskaya oblast"
        RUVLA: Vladimirskaya oblast"
        RUVGG: Volgogradskaya oblast"
        RUVLG: Vologodskaya oblast"
        RUVOR: Voronezhskaya oblast"
        RUYAR: Yaroslavskaya oblast"
        RUMOW: Moskva (autonomous city)
        RUSPE: Sankt-Peterburg (autonomous city)
        RUYEV: Yevreyskaya avtonomnaya oblast"
        RUCHU: Chukotskiy avtonomnyy okrug
        RUKHM: Khanty-Mansiyskiy avtonomnyy okrug-Yugra
        RUNEN: Nenetskiy avtonomnyy okrug
        RUYAN: Yamalo-Nenetskiy avtonomnyy okrug
      flag: "https://upload.wikimedia.org/wikipedia/en/f/f3/Flag_of_Russia.svg"
      fiscal_year: 0101-1231
      gov_fiscal_year: 0101-1231

  - code: RWA
    name: Rwanda
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/1/17/Flag_of_Rwanda.svg"

  - code: SHN
    name: Saint Helena, Ascension and Tristan da Cunha
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/en/a/ae/Flag_of_the_United_Kingdom.svg"

  - code: KNA
    name: Saint Kitts and Nevis
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/f/fe/Flag_of_Saint_Kitts_and_Nevis.svg"

  - code: LCA
    name: Saint Lucia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/9/9f/Flag_of_Saint_Lucia.svg"

  - code: SPM
    name: Saint Pierre and Miquelon
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/en/c/c3/Flag_of_France.svg"

  - code: VCT
    name: Saint Vincent and the Grenadines
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/6/6d/Flag_of_Saint_Vincent_and_the_Grenadines.svg"

  - code: BLM
    name: Saint-Barthelemy
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/d/df/Flag_of_Saint_Barthelemy_%28local%29.svg"

  - code: MAF
    name: Saint-Martin
    metadata:
      states: ~
      flag:

  - code: WSM
    name: Samoa
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/3/31/Flag_of_Samoa.svg"

  - code: SMR
    name: San Marino
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/b/b1/Flag_of_San_Marino.svg"

  - code: STP
    name: São Tomé and Príncipe
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/4/4f/Flag_of_Sao_Tome_and_Principe.svg"

  - code: SAU
    name: Saudi Arabia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/0d/Flag_of_Saudi_Arabia.svg"

  - code: SEN
    name: Senegal
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/f/fd/Flag_of_Senegal.svg"

  - code: SRB
    name: Serbia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/f/ff/Flag_of_Serbia.svg"

  - code: SYC
    name: Seychelles
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/f/fc/Flag_of_Seychelles.svg"

  - code: SLE
    name: Sierra Leone
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/1/17/Flag_of_Sierra_Leone.svg"

  - code: SGP
    name: Singapore
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/4/48/Flag_of_Singapore.svg"

  - code: SXM
    name: Sint Maarten
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/d/d3/Flag_of_Sint_Maarten.svg"

  - code: SVK
    name: Slovakia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/e/e6/Flag_of_Slovakia.svg"

  - code: SVN
    name: Slovenia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/f/f0/Flag_of_Slovenia.svg"

  - code: SLB
    name: Solomon Islands
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/7/74/Flag_of_the_Solomon_Islands.svg"

  - code: SOM
    name: Somalia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/a/a0/Flag_of_Somalia.svg"

  - code: ZAF
    name: South Africa
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/a/af/Flag_of_South_Africa.svg"

  - code: SGS
    name: South Georgia and the South Sandwich Islands
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/e/ed/Flag_of_South_Georgia_and_the_South_Sandwich_Islands.svg"

  - code: KOR
    name: South Korea
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/09/Flag_of_South_Korea.svg"
      fiscal_year: 0101-1231
      gov_fiscal_year: 0101-1231

  - code: SSD
    name: South Sudan
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/7/7a/Flag_of_South_Sudan.svg"

  - code: ESP
    name: Spain
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/en/9/9a/Flag_of_Spain.svg"
      fiscal_year: 0101-1231
      gov_fiscal_year: 0101-1231

  - code: LKA
    name: Sri Lanka
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/1/11/Flag_of_Sri_Lanka.svg"

  - code: SDN
    name: Sudan
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/01/Flag_of_Sudan.svg"

  - code: SUR
    name: Suriname
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/6/60/Flag_of_Suriname.svg"

  - code: SJM
    name: Svalbard and Jan Mayen
    metadata:
      states: ~
      flag:

  - code: SWZ
    name: Swaziland
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/f/fb/Flag_of_Eswatini.svg"

  - code: SWE
    name: Sweden
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/en/4/4c/Flag_of_Sweden.svg"
      fiscal_year: 0101-1231
      gov_fiscal_year: 0101-1231

  - code: CHE
    name: Switzerland
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/08/Flag_of_Switzerland_%28Pantone%29.svg"
      fiscal_year: 0101-1231
      gov_fiscal_year: 0101-1231

  - code: SYR
    name: Syria
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/5/53/Flag_of_Syria.svg"

  - code: TWN
    name: Taiwan
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/7/72/Flag_of_the_Republic_of_China.svg"
      fiscal_year: 0101-1231
      gov_fiscal_year: 0101-1231

  - code: TJK
    name: Tajikistan
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/d/d0/Flag_of_Tajikistan.svg"

  - code: TZA
    name: Tanzania
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/3/38/Flag_of_Tanzania.svg"

  - code: THA
    name: Thailand
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/a/a9/Flag_of_Thailand.svg"

  - code: TGO
    name: Togo
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/6/68/Flag_of_Togo.svg"

  - code: TKL
    name: Tokelau
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/8/8e/Flag_of_Tokelau.svg"

  - code: TON
    name: Tonga
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/9/9a/Flag_of_Tonga.svg"

  - code: TTO
    name: Trinidad and Tobago
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/6/64/Flag_of_Trinidad_and_Tobago.svg"

  - code: TUN
    name: Tunisia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/c/ce/Flag_of_Tunisia.svg"

  - code: TUR
    name: Turkey
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/b/b4/Flag_of_Turkey.svg"

  - code: TKM
    name: Turkmenistan
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/1/1b/Flag_of_Turkmenistan.svg"

  - code: TCA
    name: Turks and Caicos Islands
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/a/a0/Flag_of_the_Turks_and_Caicos_Islands.svg"

  - code: TUV
    name: Tuvalu
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/3/38/Flag_of_Tuvalu.svg"

  - code: UGA
    name: Uganda
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/4/4e/Flag_of_Uganda.svg"

  - code: UKR
    name: Ukraine
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/4/49/Flag_of_Ukraine.svg"

  - code: ARE
    name: United Arab Emirates
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/c/cb/Flag_of_the_United_Arab_Emirates.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231

  - code: GBR
    name: United Kingdom
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/en/a/ae/Flag_of_the_United_Kingdom.svg"
      fiscal_year: 0406-0405
      gov_fiscal_year: 0401-0331

  - code: UMI
    name: United States Minor Outlying Islands
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/en/a/a4/Flag_of_the_United_States.svg"

  - code: URY
    name: Uruguay
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/f/fe/Flag_of_Uruguay.svg"

  - code: VIR
    name: US Virgin Islands
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/f/f8/Flag_of_the_United_States_Virgin_Islands.svg"

  - code: USA
    name: USA
    metadata:
      states:
        USAK: Alaska
        USAL: Alabama
        USAR: Arkansas
        USAZ: Arizona
        USCA: California
        USCO: Colorado
        USCT: Connecticut
        USDC: Washington D.C.
        USDE: Delaware
        USFL: Florida
        USGA: Georgia
        USHI: Hawaii
        USIA: Iowa
        USID: Idaho
        USIL: Illinois
        USIN: Indiana
        USKS: Kansas
        USKY: Kentucky
        USLA: Louisiana
        USMA: Massachusetts
        USMD: Maryland
        USME: Maine
        USMI: Michigan
        USMN: Minnesota
        USMO: Missouri
        USMS: Mississippi
        USMT: Montana
        USNC: North Carolina
        USND: North Dakota
        USNE: Nebraska
        USNH: New Hampshire
        USNJ: New Jersey
        USNM: New Mexico
        USNV: Nevada
        USNY: New York
        USOH: Ohio
        USOK: Oklahoma
        USOR: Oregon
        USPA: Pennsylvania
        USRI: Rhode Island
        USSC: South Carolina
        USSD: South Dakota
        USTN: Tennessee
        USTX: Texas
        USUT: Utah
        USVA: Virginia
        USVT: Vermont
        USWA: Washington
        USWI: Wisconsin
        USWV: West Virginia
        USWY: Wyoming
        USAS: American Samoa
        USGU: Guam
        USMP: Northern Mariana Islands
        USPR: Puerto Rico
        USUM: United States Minor Outlying Islands
        USVI: US Virgin Islands
      flag: "https://upload.wikimedia.org/wikipedia/en/a/a4/Flag_of_the_United_States.svg"

  - code: UZB
    name: Uzbekistan
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/8/84/Flag_of_Uzbekistan.svg"

  - code: VUT
    name: Vanuatu
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/6/6e/Flag_of_Vanuatu_%28official%29.svg"

  - code: VAT
    name: Vatican City
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/00/Flag_of_the_Vatican_City.svg"

  - code: VEN
    name: Venezuela
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/7/7b/Flag_of_Venezuela_%28state%29.svg"

  - code: VNM
    name: Vietnam
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/2/21/Flag_of_Vietnam.svg"

  - code: WLF
    name: Wallis and Futuna
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/en/c/c3/Flag_of_France.svg"

  - code: ESH
    name: Western Sahara
    metadata:
      states: ~
      flag:

  - code: YEM
    name: Yemen
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/8/89/Flag_of_Yemen.svg"

  - code: ZMB
    name: Zambia
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/0/06/Flag_of_Zambia.svg"

  - code: ZWE
    name: Zimbabwe
    metadata:
      states: ~
      flag: "https://upload.wikimedia.org/wikipedia/commons/6/6a/Flag_of_Zimbabwe.svg"

`

// Rejections - Code/Text combinations returned in rejection messages when
// a request is not accepted.
var yamlRejections = `

  - code: 0
    name: Success
    label: Success
    description: No failure. This code should not be used in a reject message.

  - code: 1
    name: MsgMalformed
    label: Message Malformed
    description: The OP_RETURN message is malformed. It doesn't pass data validation or something about it isn't according to protocol.

  - code: 2
    name: TxMalformed
    label: Transaction Malformed
    description: The Bitcoin tx is malformed. Incorrect inputs/outputs or something similar.

  - code: 3
    name: Timeout
    label: Time Out
    description: A dependency, other contract/service, has failed to complete before the smart contract's timeout.

#################################### Contract ####################################

  - code: 10
    name: ContractExists
    label: Contract Already Exists
    description: The contract already exists and can't be recreated.

  - code: 11
    name: ContractNotDynamic
    label: Contract Is Not Dynamic
    description: 'Sent when a CO is received, but the contract type is not "D" (Dynamic)'

  - code: 12
    name: ContractAssetQtyReduction
    label: Contract Asset Quantity Reduction
    description: Sent when a CA tries to reduce the number of assets below the number of assets the contract has.

  - code: 13
    name: ContractFixedQuantity
    label: Contract Fixed Quantity
    description: Sent when the issuer attempted to increase the quantity of assets in a contract beyond the fixed quantity permitted.

  - code: 14
    name: ContractAuthFlags
    label: Contract Auth Flags Prohibit
    description: The contract auth flags don't permit the action requested.

  - code: 15
    name: ContractExpired
    label: Contract Expired
    description: The contract is expired so can no longer accept requests.

  - code: 16
    name: ContractFrozen
    label: Contract Frozen
    description: The contract is frozen and the request is not permitted while frozen.

  - code: 17
    name: ContractRevision
    label: Contract Revision Incorrect
    description: The revision in a contract amendment is incorrect.

  - code: 18
    name: ContractNotPermitted
    label: Contract Not Permitted
    description: Action not permitted by contract.

#################################### Asset ####################################

  - code: 20
    name: AssetCodeExists
    label: Asset Code Already Exists
    description: The asset code specified already exists and can't be reused.

  - code: 21
    name: AssetNotFound
    label: Asset Not Found
    description: The asset code is not found.

  - code: 22
    name: AssetAuthFlags
    label: Asset Auth Flags Prohibit
    description: The asset auth flags don't permit the action requested.

  - code: 23
    name: AssetFrozen
    label: Asset Frozen
    description: The asset is frozen and the request is not permitted while frozen.

  - code: 24
    name: AssetRevision
    label: Asset Revision Incorrect
    description: The revision in an asset amendment is incorrect.

  - code: 25
    name: AssetNotPermitted
    label: Asset Not Permitted
    description: Action not permitted by asset.

#################################### Transfer ####################################

  - code: 30
    name: TransferSelf
    label: Transfer To Self Prohibited
    description: Transfers with the sender and receiver addresses the same are not permitted.

  - code: 31
    name: TransferExpired
    label: Transfer Expired
    description: The transfer has expired.

  - code: 32
    name: HoldingsFrozen
    label: Holdings Frozen
    description: Holdings are frozen, so the request can't be completed.

#################################### Governance ####################################

  - code: 40
    name: HolderProposalProhibited
    label: Holder Proposal Prohibited
    description: Holders are not permitted to make proposals.

  - code: 41
    name: ProposalConflicts
    label: Proposal Conflicts
    description: The proposal conflicts with an unapplied proposal.

  - code: 42
    name: VoteNotFound
    label: Vote Not Found
    description: The vote ID referenced is not found.

  - code: 43
    name: VoteClosed
    label: Vote Closed
    description: The vote has closed and ballots are no longer permitted.

  - code: 44
    name: BallotAlreadyCounted
    label: Ballot Already Counted
    description: The ballot has already been counted for this address.

  - code: 45
    name: VoteSystemNotPermitted
    label: Vote System Not Permitted
    description: "The voting system isn't permitted for this request."

#################################### Enforcement ####################################


#################################### Funding ####################################

  - code: 60
    name: InsufficientTxFeeFunding
    label: Insufficient Transaction Fee Funding
    description: Insufficient bitcoin quantities for response transaction fees.

  - code: 61
    name: InsufficientValue
    label: Insufficient Value
    description: Insufficient bitcoin quantity in inputs to fund request.

  - code: 62
    name: InsufficientQuantity
    label: Insufficient Quantity
    description: Insufficient token holdings to for request.

#################################### Address ####################################

  - code: 70
    name: NotIssuer
    label: Requestor Is Not Issuer
    description: The requestor is not the issuer and is required to be for this request.

  - code: 71
    name: NotOperator
    label: Requestor Is Not Operator
    description: The requestor is not the operator and is required to be for this request.

  - code: 72
    name: UnauthorizedAddress
    label: Unauthorized Address
    description: The address specified is not permitted for this request.

#################################### Signatures ####################################

  - code: 80
    name: InvalidSignature
    label: Invalid Signature
    description: The signature provided is not valid. This is for signatures included within OP_RETURN data. Not bitcoin transaction signature scripts.

`

// Roles - Roles that entities play in relation to their interactions with
// other entities. These roles have widely-accepted tasks, rights and
// duties.
var yamlRoles = `
  - code: 1
    label: Accountant

  - code: 2
    label: Advisor

  - code: 3
    label: Agent

  - code: 4
    label: Beneficiary

  - code: 5
    label: CEO

  - code: 6
    label: CFO

  - code: 7
    label: Chair

  - code: 8
    label: COO

  - code: 9
    label: CTO

  - code: 10
    label: Custodian

  - code: 11
    label: Director

  - code: 12
    label: Executive

  - code: 13
    label: Lawyer

  - code: 14
    label: Legal Guardian

  - code: 15
    label: Limited Partner

  - code: 16
    label: Manager

  - code: 17
    label: Managing Partner

  - code: 18
    label: Member
    description: Shareholder

  - code: 19
    label: Partner

  - code: 20
    label: Principal

  - code: 21
    label: Proprietor

  - code: 22
    label: Protector

  - code: 23
    label: Secretary

  - code: 24
    label: Settlor

  - code: 25
    label: Significant Member
    description: Major Shareholder

  - code: 26
    label: Smart Contract Operator

  - code: 27
    label: Trader

  - code: 28
    label: Trustee

  - code: 29
    label: Unit Holder

`
