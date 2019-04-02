package protocol

import (
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

type RejectionCode struct {
	Name        string
	Text        string
	Description string
}

var rejectionCodes map[uint8]RejectionCode

func GetRejectionCodes() (map[uint8]RejectionCode, error) {
	if rejectionCodes != nil {
		return rejectionCodes, nil
	}

	load := struct {
		Values map[uint8]RejectionCode
	}{}

	if err := yaml.Unmarshal([]byte(yamlRejectionCodes), &load); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal rejection codes yaml")
	}

	rejectionCodes = load.Values
	return rejectionCodes, nil
}

type CurrencyType struct {
	Name              string
	Label             string
	Symbol            string
	Fractions         uint64
	FractionalUnit    string `yaml:"fractional_unit"`
	Polity            string
	MonetaryAuthority string `yaml:"monetary_authority"`
}

var currencies map[string]CurrencyType

func GetCurrencies() (map[string]CurrencyType, error) {
	if currencies != nil {
		return currencies, nil
	}

	load := struct {
		Values map[string]CurrencyType
	}{}

	if err := yaml.Unmarshal([]byte(yamlCurrencies), &load); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal currencies yaml")
	}

	currencies = load.Values
	return currencies, nil
}

type EntityType struct {
	Name   string
	Label  string
	Symbol string
}

var entities map[string]EntityType

func GetEntities() (map[string]EntityType, error) {
	if entities != nil {
		return entities, nil
	}

	load := struct {
		Values map[string]EntityType
	}{}

	if err := yaml.Unmarshal([]byte(yamlEntities), &load); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal entities yaml")
	}

	entities = load.Values
	return entities, nil
}

type Polity struct {
	Name          string
	States        map[string]string
	Flag          string
	FiscalYear    string `yaml:"fractional_unit"`
	GovFiscalYear string `yaml:"gov_fiscal_year"`
}

var polities map[string]Polity

func GetPolities() (map[string]Polity, error) {
	if polities != nil {
		return polities, nil
	}

	load := struct {
		Values map[string]Polity
	}{}

	if err := yaml.Unmarshal([]byte(yamlPolities), &load); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal polities yaml")
	}

	polities = load.Values
	return polities, nil
}

type MessageType struct {
	Name string
}

var messages map[uint16]MessageType

func GetMessages() (map[uint16]MessageType, error) {
	if messages != nil {
		return messages, nil
	}

	load := struct {
		Values map[uint16]MessageType
	}{}

	if err := yaml.Unmarshal([]byte(yamlMessages), &load); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal messages yaml")
	}

	messages = load.Values
	return messages, nil
}

type RoleType struct {
	Name string
}

var roles map[uint8]RoleType

func GetRoles() (map[uint8]RoleType, error) {
	if roles != nil {
		return roles, nil
	}

	load := struct {
		Values map[uint8]RoleType
	}{}

	if err := yaml.Unmarshal([]byte(yamlRoles), &load); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal roles yaml")
	}

	roles = load.Values
	return roles, nil
}

// Currencies - Currencies
var yamlCurrencies = `

metadata:
  name: Currencies
  description: "Currencies"
  type: CurrencyType


values:
  AED:
    name: UnitedArabEmiratesDirham
    label: United Arab Emirates dirham
    symbol: "د.إ"
    fractions: 100
    fractional_unit: Fils
    polity: ARE
    monetary_authority: Central Bank of the United Arab Emirates
  AFN:
    name: AfghanAfghani
    label: Afghan afghani
    symbol: "؋"
    fractions: 100
    fractional_unit: Pul
    polity: AFG
    monetary_authority: Central Bank of Afghanistan
  ALL:
    name: AlbanianLek
    label: Albanian lek
    symbol: "L"
    fractions: 100
    fractional_unit: Qindarkë
    polity: ALB
    monetary_authority: Bank of Albania
  AMD:
    name: ArmenianDram
    label: Armenian dram
    symbol: "֏"
    fractions: 100
    fractional_unit: Luma
    polity: ARM
    monetary_authority: Central Bank of the Republic of Armenia
  ANG:
    name: NetherlandsAntilleanGuilder
    label: Netherlands Antillean guilder
    symbol: "ƒ"
    fractions: 100
    fractional_unit: Cent
    polity: SXM, CUW
    monetary_authority: Central Bank of Curaçao and Sint Maarten
  AOA:
    name: AngolanKwanza
    label: Angolan kwanza
    symbol: "Kz"
    fractions: 100
    fractional_unit: Cêntimo
    polity: AGO
    monetary_authority: Central Bank of Angola
  ARS:
    name: ArgentinePeso
    label: Argentine peso
    symbol: "$"
    fractions: 100
    fractional_unit: Centavo
    polity: ARG
    monetary_authority: Central Bank of Argentina
  AUD:
    name: AustralianDollar
    label: Australian dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: AUS, NRU, KIR, TUV
    monetary_authority: Reserve Bank of Australia
  AWG:
    name: ArubanFlorin
    label: Aruban florin
    symbol: "ƒ"
    fractions: 100
    fractional_unit: Cent
    polity: ABW
    monetary_authority: Central Bank of Aruba
  AZN:
    name: AzerbaijaniManat
    label: Azerbaijani manat
    symbol: "₼"
    fractions: 100
    fractional_unit: Qəpik
    polity: AZE
    monetary_authority: Central Bank of Azerbaijan
  BAM:
    name: BosniaHerzegovinaConvertibleMark
    label: Bosnia and Herzegovina convertible mark
    symbol: "КМ"
    fractions: 100
    fractional_unit: Fening
    polity: BIH
    monetary_authority: Central Bank of Bosnia and Herzegovina
  BBD:
    name: BarbadianDollar
    label: Barbadian dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: BRB
    monetary_authority: Central Bank of Barbados
  BDT:
    name: BangladeshiTaka
    label: Bangladeshi taka
    symbol: "৳"
    fractions: 100
    fractional_unit: Poisha
    polity: BGD
    monetary_authority: Bangladesh Bank
  BGN:
    name: BulgarianLev
    label: Bulgarian lev
    symbol:	"лв"
    fractions: 100
    fractional_unit: Stotinka
    polity: BGR
    monetary_authority: Bulgarian National Bank
  BHD:
    name: BahrainiDinar
    label: Bahraini dinar
    symbol:	".د.ب"
    fractions: 1000
    fractional_unit: Fils
    polity: BHR
    monetary_authority: Central Bank of Bahrain
  BIF:
    name: BurundianFranc
    label: Burundian franc
    symbol: "Fr"
    fractions: 100
    fractional_unit: Centime
    polity: BDI
    monetary_authority: Bank of the Republic of Burundi
  BMD:
    name: BermudianDollar
    label: Bermudian dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: BMU
    monetary_authority: Bermuda Monetary Authority
  BND:
    name: BruneiDollar
    label: Brunei dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Sen
    polity: BRN
    monetary_authority: Monetary Authority of Brunei Darussalam
  BOB:
    name: BolivianBoliviano
    label: Bolivian boliviano
    symbol: "Bs."
    fractions: 100
    fractional_unit: Centavo
    polity: BOL
    monetary_authority: Central Bank of Bolivia
  BRL:
    name: BrazilianReal
    label: Brazilian real
    symbol: "R$"
    fractions: 100
    fractional_unit: Centavo
    polity: BRA
    monetary_authority: Central Bank of Brazil
  BSD:
    name: BahamianDollar
    label: Bahamian dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: BHS
    monetary_authority: Central Bank of The Bahamas
  BSV:
    name: Bitcoin
    label: Bitcoin SV
    symbol: "₿"
    fractions: 100000000
    fractional_unit: Satoshis
    polity: null
    monetary_authority: Satoshi Nakamoto
  BTN:
    name: BhutaneseNgultrum
    label: Bhutanese ngultrum
    symbol: "Nu."
    fractions: 100
    fractional_unit: Chetrum
    polity: BTN
    monetary_authority: Royal Monetary Authority of Bhutan
  BWP:
    name: BotswanaPula
    label: Botswana pula
    symbol: "P"
    fractions: 100
    fractional_unit: Thebe
    polity: BWA
    monetary_authority: Bank of Botswana
  BYN:
    name: BelarusianRuble
    label: Belarusian ruble
    symbol: "Br"
    fractions: 100
    fractional_unit: Kapyeyka
    polity: BLR
    monetary_authority: National Bank of the Republic of Belarus
  BZD:
    name: BelizeDollar
    label: Belize dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: BLZ
    monetary_authority: Central Bank of Belize
  CAD:
    name: CanadianDollar
    label: Canadian dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: CAN
    monetary_authority: Bank of Canada
  CDF:
    name: CongoleseFranc
    label: Congolese franc
    symbol: "Fr"
    fractions: 100
    fractional_unit: Centime
    polity: COD
    monetary_authority: Central Bank of the Congo
  CHF:
    name: SwissFranc
    label: Swiss franc
    symbol: "Fr"
    fractions: 100
    fractional_unit: Rappen
    polity: CHE, LIE
    monetary_authority: Swiss National Bank
  CLP:
    name: ChileanPeso
    label: Chilean peso
    symbol: "$"
    fractions: 100
    fractional_unit: Centavo
    polity: CHL
    monetary_authority: Central Bank of Chile
  CNY:
    name: ChineseYuan
    label: Chinese yuan
    symbol: "¥"
    fractions: 100
    fractional_unit: Fen
    polity: CHN
    monetary_authority: People's Bank of China
  COP:
    name: ColombianPeso
    label: Colombian peso
    symbol: "$"
    fractions: 100
    fractional_unit: Centavo
    polity: COL
    monetary_authority: Bank of the Republic
  CRC:
    name: CostaRicanColon
    label: Costa Rican colon
    symbol: "₡"
    fractions: 100
    fractional_unit: Céntimo
    polity: CRI
    monetary_authority: Central Bank of Costa Rica
  CUC:
    name: CubanConvertiblePeso
    label: Cuban convertible peso
    symbol: "$"
    fractions: 100
    fractional_unit: Centavo
    polity: CUB
    monetary_authority: Central Bank of Cuba
  CUP:
    name: CubanConvertiblePeso
    label: Cuban convertible peso
    symbol: "$"
    fractions: 100
    fractional_unit: Centavo
    polity: CUB
    monetary_authority: Central Bank of Cuba
  CVE:
    name: CapeVerdeanEscudo
    label: Cape Verdean escudo
    symbol: "$"
    fractions: 100
    fractional_unit: Centavo
    polity: CUB
    monetary_authority: Bank of Cape Verde
  CZK:
    name: CzechKoruna
    label: Czech koruna
    symbol: "Kč"
    fractions: 100
    fractional_unit: Haléř
    polity: CZE
    monetary_authority: Czech National Bank
  DJF:
    name: DjiboutianFranc
    label: Djiboutian franc
    symbol: "Fr"
    fractions: 100
    fractional_unit: Centime
    polity: DJI
    monetary_authority: Central Bank of Djibouti
  DKK:
    name: DanishKrone
    label: Danish krone
    symbol: "kr"
    fractions: 100
    fractional_unit: Øre
    polity: DNK,FRO, GRL
    monetary_authority: Danmarks Nationalbank
  DOP:
    name: DominicanPeso
    label: Dominican peso
    symbol: "$"
    fractions: 100
    fractional_unit: Centavo
    polity: DOM
    monetary_authority: Central Bank of the Dominican Republic
  DZD:
    name: AlgerianDinar
    label: Algerian dinar
    symbol: "د.ج"
    fractions: 100
    fractional_unit: Santeem
    polity: DZA
    monetary_authority: Bank of Algeria
  EGP:
    name: EgyptianPound
    label: Egyptian pound
    symbol: "E£"
    fractions: 100
    fractional_unit: Piastre
    polity: EGY
    monetary_authority: Central Bank of Egypt
  ERN:
    name: EritreanNakfa
    label: Eritrean nakfa
    symbol: "Nfk"
    fractions: 100
    fractional_unit: Piastre
    polity: ERI
    monetary_authority: Bank of Eritrea
  ETB:
    name: EthiopianBirr
    label: Ethiopian birr
    symbol: "Br"
    fractions: 100
    fractional_unit: Santim
    polity: ETH
    monetary_authority: Bank of Eritrea
  EUR:
    name: Euro
    label: Euro
    symbol: "€"
    fractions: 100
    fractional_unit: Cent
    polity: EU, ALA, AND, AUT, BEL, CYP, EST, FIN, FRA, ATF, DEU, GRC, GLP, IRL, ITA, LVA, LTU, LUX, MLT, GUF, MTQ, MYT, MCO, MNE, NLD, PRT, REU, BLM, MAF, SPM, SMR, SVK, SVN, ESP
    monetary_authority: European Central Bank
  FJD:
    name: FijianDollar
    label: Fijian dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: FJI
    monetary_authority: Reserve Bank of Fiji
  FKP:
    name: FalklandIslandsPound
    label: Falkland Islands pound
    symbol: "£"
    fractions: 100
    fractional_unit: Penny
    polity: FLK
    monetary_authority: Government of the Falkland Islands
  GBP:
    name: PoundSterling
    label: Pound sterling
    symbol: "£"
    fractions: 100
    fractional_unit: Penny
    polity: GBR, IOT, IMN, JEY, GGY
    monetary_authority: Bank of England
  GEL:
    name: GeorgianLari
    label: Georgian lari
    symbol: "₾"
    fractions: 100
    fractional_unit: Tetri
    polity: GEO
    monetary_authority: National Bank of Georgia
  GHS:
    name: GhanaianCedi
    label: Ghanaian cedi
    symbol: "₵"
    fractions: 100
    fractional_unit: Pesewa
    polity: GHA
    monetary_authority: Bank of Ghana
  GIP:
    name: GibraltarPound
    label: Gibraltar pound
    symbol: "£"
    fractions: 100
    fractional_unit: Penny
    polity: GIB
    monetary_authority: Government of Gibraltar
  GMD:
    name: GambianDalasi
    label: Gambian dalasi
    symbol: "D"
    fractions: 100
    fractional_unit: Butut
    polity: GMB
    monetary_authority: Central Bank of The Gambia
  GNF:
    name: GuineanFranc
    label: Guinean franc
    symbol: "Fr"
    fractions: 100
    fractional_unit: Centime
    polity: GNQ
    monetary_authority: Government of Gibraltar
  GTQ:
    name: GuatemalanQuetzal
    label: Guatemalan quetzal
    symbol: "Q"
    fractions: 100
    fractional_unit: Centavo
    polity: GTM
    monetary_authority: Bank of Guatemala
  GYD:
    name: GuyaneseDollar
    label: Guyanese dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: GUY
    monetary_authority: Bank of Guyana
  HKD:
    name: HongKongDollar
    label: Hong Kong dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: HKG
    monetary_authority: Hong Kong Monetary Authority
  HNL:
    name: HonduranLempira
    label: Honduran lempira
    symbol: "L"
    fractions: 100
    fractional_unit: Centavo
    polity: HND
    monetary_authority: Central Bank of Honduras
  HRK:
    name: CroatianKuna
    label: Croatian kuna
    symbol: "kn"
    fractions: 100
    fractional_unit: Lipa
    polity: HRV
    monetary_authority: Croatian National Bank
  HTG:
    name: HaitianGourde
    label: Haitian gourde
    symbol: "G"
    fractions: 100
    fractional_unit: 	Centime
    polity: HTI
    monetary_authority: Bank of the Republic of Haiti
  HUF:
    name: HungarianForint
    label: Hungarian forint
    symbol: "Ft"
    fractions: 100
    fractional_unit: 	Fillér
    polity: HUN
    monetary_authority: Hungarian National Bank
  IDR:
    name: IndonesianRupiah
    label: Indonesian rupiah
    symbol: "Rp"
    fractions: 100
    fractional_unit: 	Sen
    polity: IDN
    monetary_authority: Bank of Indonesia
  ILS:
    name: IsraeliNewShekel
    label: Israeli new shekel
    symbol: "₪"
    fractions: 100
    fractional_unit: 	Agora
    polity: ISR, PSE
    monetary_authority: Bank of Israel
  INR:
    name: IndianRupee
    label: Indian rupee
    symbol: "₹"
    fractions: 100
    fractional_unit: 	Paisa
    polity: IND, BTN
    monetary_authority: Reserve Bank of India
  IQD:
    name: IraqiDinar
    label: Iraqi dinar
    symbol: "ع.د"
    fractions: 1000
    fractional_unit: 	Fils
    polity: IRQ
    monetary_authority: Central Bank of Iraq
  IRR:
    name: IranianRial
    label: Iranian rial
    symbol: "﷼"
    fractions: 100
    fractional_unit: Dinar
    polity: IRN
    monetary_authority: Central Bank of Iraq
  ISK:
    name: IcelandicKrona
    label: Icelandic króna
    symbol: "kr"
    fractions: 100
    fractional_unit: Eyrir
    polity: ISL
    monetary_authority: Central Bank of Iceland
  JMD:
    name: JamaicanDollar
    label: Jamaican dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: JAM
    monetary_authority: Bank of Jamaica
  JOD:
    name: JordanianDinar
    label: Jordanian dinar
    symbol: "د.ا"
    fractions: 100
    fractional_unit: Piastre
    polity: JOR, PSE
    monetary_authority: Central Bank of Jordan
  JPY:
    name: JapaneseYen
    label: Japanese yen
    symbol: "¥"
    fractions: 100
    fractional_unit: Sen
    polity: JPN
    monetary_authority: Bank of Japan
  KES:
    name: KenyanShilling
    label: Kenyan shilling
    symbol: "Sh"
    fractions: 100
    fractional_unit: Cent
    polity: KEN
    monetary_authority: Central Bank of Kenya
  KGS:
    name: KyrgyzstaniSom
    label: Kyrgyzstani som
    symbol: "с"
    fractions: 100
    fractional_unit: Tyiyn
    polity: KGZ
    monetary_authority: National Bank of the Kyrgyz Republic
  KHR:
    name: CambodianRiel
    label: Cambodian riel
    symbol: "៛"
    fractions: 100
    fractional_unit: Sen
    polity: KHM
    monetary_authority: National Bank of Cambodia
  KMF:
    name: ComoroFranc
    label: Comoro franc
    symbol: "Fr"
    fractions: 100
    fractional_unit: Sen
    polity: COM
    monetary_authority: Central Bank of the Comoros
  KPW:
    name: NorthKoreanWon
    label: North Korean won
    symbol: "₩"
    fractions: 100
    fractional_unit: Chon
    polity: PRK
    monetary_authority: Central Bank of the Democratic People's Republic of Korea
  KRW:
    name: SouthKoreanWon
    label: South Korean won
    symbol: "₩"
    fractions: 100
    fractional_unit: Jeon
    polity: KOR
    monetary_authority: Bank of Korea
  KWD:
    name: KuwaitiDinar
    label: Kuwaiti dinar
    symbol: "د.ك"
    fractions: 1000
    fractional_unit: Fils
    polity: KWT
    monetary_authority: Central Bank of Kuwait
  KYD:
    name: CaymanIslandsDollar
    label: Cayman Islands dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: CYM
    monetary_authority: Cayman Islands Monetary Authority
  KZT:
    name: KazakhstaniTenge
    label: Kazakhstani tenge
    symbol: "₸"
    fractions: 100
    fractional_unit: Tïın
    polity: KAZ
    monetary_authority: National Bank of Kazakhstan
  LAK:
    name: LaoKip
    label: Lao kip
    symbol: "₭"
    fractions: 100
    fractional_unit: Att
    polity: LAO
    monetary_authority: Bank of the Lao People's Democratic Republic
  LBP:
    name: LebanesePound
    label: Lebanese pound
    symbol: "ل.ل"
    fractions: 100
    fractional_unit: Piastre
    polity: LBN
    monetary_authority: Banque du Liban
  LKR:
    name: SriLankanRupee
    label: Sri Lankan rupee
    symbol: "Rs"
    fractions: 100
    fractional_unit: Cent
    polity: LKA
    monetary_authority: Central Bank of Sri Lanka
  LRD:
    name: LiberianDollar
    label: Liberian dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: LBR
    monetary_authority: Central Bank of Liberia
  LSL:
    name: LesothoLoti
    label: Lesotho loti
    symbol: "L"
    fractions: 100
    fractional_unit: Sente
    polity: LBR
    monetary_authority: Central Bank of Lesotho
  LYD:
    name: LibyanDinar
    label: Libyan dinar
    symbol: "ل.د"
    fractions: 1000
    fractional_unit: Dirham
    polity: LBY
    monetary_authority: Central Bank of Libya
  MAD:
    name: MoroccanDirham
    label: Moroccan dirham
    symbol: "د.م."
    fractions: 100
    fractional_unit: Centime
    polity: MAR
    monetary_authority: Bank Al-Maghrib
  MDL:
    name: MoldovanLeu
    label: Moldovan leu
    symbol: "L"
    fractions: 100
    fractional_unit: Ban
    polity: MDA
    monetary_authority: National Bank of Moldova
  MGA:
    name: MalagasyAriary
    label: Malagasy ariary
    symbol: "Ar"
    fractions: 5
    fractional_unit: Iraimbilanja
    polity: MDG
    monetary_authority: Central Bank of Madagascar
  MKD:
    name: MacedonianDenar
    label: Macedonian denar
    symbol: "ден"
    fractions: 100
    fractional_unit: Deni
    polity: MKD
    monetary_authority: National Bank of the Republic of Macedonia
  MMK:
    name: MyanmarKyat
    label: Myanmar kyat
    symbol: "Ks"
    fractions: 100
    fractional_unit: Pya
    polity: MMR
    monetary_authority: Central Bank of Myanmar
  MNT:
    name: MongolianTogrog
    label: Mongolian tögrög
    symbol: "₮"
    fractions: 100
    fractional_unit: Möngö
    polity: MNG
    monetary_authority: Bank of Mongolia
  MOP:
    name: MacanesePataca
    label: Macanese pataca
    symbol: "P"
    fractions: 100
    fractional_unit: Avo
    polity: MAC
    monetary_authority: Monetary Authority of Macau
  MRU:
    name: MauritanianOuguiya
    label: Mauritanian ouguiya
    symbol: "UM"
    fractions: 5
    fractional_unit: Khoums
    polity: MRT
    monetary_authority: Central Bank of Mauritania
  MUR:
    name: MauritianRupee
    label: Mauritian rupee
    symbol: "₨"
    fractions: 100
    fractional_unit: Cent
    polity: MUS
    monetary_authority: Bank of Mauritius
  MVR:
    name: MaldivianRufiyaa
    label: Maldivian rufiyaa
    symbol: ".ރ"
    fractions: 100
    fractional_unit: Laari
    polity: MDV
    monetary_authority: Maldives Monetary Authority
  MWK:
    name: MalawianKwacha
    label: Malawian kwacha
    symbol: "MK"
    fractions: 100
    fractional_unit: Tambala
    polity: MWI
    monetary_authority: Reserve Bank of Malawi
  MXN:
    name: MexicanPeso
    label: Mexican peso
    symbol: "$"
    fractions: 100
    fractional_unit: Centavo
    polity: MEX
    monetary_authority: Bank of Mexico
  MYR:
    name: MalaysianRinggit
    label: Malaysian ringgit
    symbol: "RM"
    fractions: 100
    fractional_unit: Sen
    polity: MYS
    monetary_authority: Central Bank of Malaysia
  MZN:
    name: MozambicanMetical
    label: Mozambican metical
    symbol: "MT"
    fractions: 100
    fractional_unit: Centavo
    polity: MOZ
    monetary_authority: Bank of Mozambique
  NAD:
    name: NamibianDollar
    label: Namibian dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: NAM
    monetary_authority: Bank of Namibia
  NGN:
    name: NigerianNaira
    label: Nigerian naira
    symbol: "₦"
    fractions: 100
    fractional_unit: Kobo
    polity: NGA
    monetary_authority: Central Bank of Nigeria
  NIO:
    name: NicaraguaCordoba
    label: Nicaraguan córdoba
    symbol: "C$"
    fractions: 100
    fractional_unit: Centavo
    polity: NIC
    monetary_authority: Central Bank of Nicaragua
  NOK:
    name: NorwegianKrone
    label: Norwegian krone
    symbol: "kr"
    fractions: 100
    fractional_unit: Øre
    polity: NOR, SJM, BVT
    monetary_authority: Norges Bank
  NPR:
    name: NepaleseRupee
    label: Nepalese rupee
    symbol: "₨"
    fractions: 100
    fractional_unit: Paisa
    polity: NPL
    monetary_authority: Nepal Rastra Bank
  NZD:
    name: NewZealandDollar
    label: New Zealand dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: NZL, COK, NIU, PCN, TKL
    monetary_authority: Reserve Bank of New Zealand
  OMR:
    name: OmaniRial
    label: Omani rial
    symbol: "ر.ع."
    fractions: 1000
    fractional_unit: Baisa
    polity: OMN
    monetary_authority: Central Bank of Oman
  PAB:
    name: PanamanianBalboa
    label: Panamanian balboa
    symbol: "B/."
    fractions: 100
    fractional_unit: Centésimo
    polity: PAN
    monetary_authority: Central Bank of Oman
  PEN:
    name: PeruvianSol
    label: Peruvian sol
    symbol: "S/."
    fractions: 100
    fractional_unit: Céntimo
    polity: PER
    monetary_authority: Central Reserve Bank of Peru
  PGK:
    name: PapuaNewGuineanKina
    label: Papua New Guinean kina
    symbol: "K"
    fractions: 100
    fractional_unit: Toea
    polity: PNG
    monetary_authority: Bank of Papua New Guinea
  PHP:
    name: PhilippinePeso
    label: Philippine peso
    symbol: "₱"
    fractions: 100
    fractional_unit: Sentimo
    polity: PHL
    monetary_authority: Central Bank of the Philippines
  PKR:
    name: PakistaniRupee
    label: Pakistani rupee
    symbol: "₨"
    fractions: 100
    fractional_unit: Paisa
    polity: PAK
    monetary_authority: State Bank of Pakistan
  PLN:
    name: PolishZloty
    label: Polish złoty
    symbol: "zł"
    fractions: 100
    fractional_unit: Grosz
    polity: POL
    monetary_authority: National Bank of Poland
  PYG:
    name: ParaguayanGuarani
    label: Paraguayan guaraní
    symbol: "₲"
    fractions: 100
    fractional_unit: Céntimo
    polity: PRY
    monetary_authority: Central Bank of Paraguay
  QAR:
    name: QatariRiyal
    label: Qatari riyal
    symbol: "ر.ق"
    fractions: 100
    fractional_unit: Dirham
    polity: QAT
    monetary_authority: Qatar Central Bank
  RON:
    name: RomanianLeu
    label: Romanian leu
    symbol: "lei"
    fractions: 100
    fractional_unit: Ban
    polity: ROU
    monetary_authority: National Bank of Romania
  RSD:
    name: SerbianDinar
    label: Serbian dinar
    symbol: "din"
    fractions: 100
    fractional_unit: Para
    polity: SRB
    monetary_authority: National Bank of Serbia
  RUB:
    name: RussianRuble
    label: Russian ruble
    symbol: "₽"
    fractions: 100
    fractional_unit: Kopek
    polity: RUS
    monetary_authority: Bank of Russia
  RWF:
    name: RwandanFranc
    label: Rwandan franc
    symbol: "Fr"
    fractions: 100
    fractional_unit: Centime
    polity: RWA
    monetary_authority: National Bank of Rwanda
  SAR:
    name: SaudiRiyal
    label: Saudi riyal
    symbol: "ر.س"
    fractions: 100
    fractional_unit: Halala
    polity: SAU
    monetary_authority: Saudi Arabian Monetary Authority
  SBD:
    name: SolomonIslandDollar
    label: Solomon Islands dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: SLB
    monetary_authority: Central Bank of Solomon Islands
  SCR:
    name: SeychelloisRupee
    label: Seychellois rupee
    symbol: "₨"
    fractions: 100
    fractional_unit: Cent
    polity: SYC
    monetary_authority: Central Bank of Solomon Islands
  SDG:
    name: SudanesePound
    label: Sudanese pound
    symbol: "ج.س."
    fractions: 100
    fractional_unit: Piastre
    polity: SSD
    monetary_authority: Bank of Sudan
  SEK:
    name: SwedishKrona
    label: Swedish krona
    symbol: "kr"
    fractions: 100
    fractional_unit: Öre
    polity: SWE
    monetary_authority: Sveriges Riksbank
  SGD:
    name: SingaporeDollar
    label: Singapore dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: SGP
    monetary_authority: Monetary Authority of Singapore
  SHP:
    name: SaintHelenaPound
    label: Saint Helena pound
    symbol: "£"
    fractions: 100
    fractional_unit: Penny
    polity: SHN
    monetary_authority: Government of Saint Helena
  SLL:
    name: SierraLeoneanLeone
    label: Sierra Leonean leone
    symbol: "Le"
    fractions: 100
    fractional_unit: Cent
    polity: SLE
    monetary_authority: Bank of Sierra Leone
  SOS:
    name: SomaliShilling
    label: Somali shilling
    symbol: "Sl"
    fractions: 100
    fractional_unit: Cent
    polity: SOM
    monetary_authority: Central Bank of Somalia
  SRD:
    name: SurilabelseDollar
    label: Surilabelse dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: SUR
    monetary_authority: Central Bank of Surilabel
  SSP:
    name: SouthSudanesePound
    label: South Sudanese pound
    symbol: "£"
    fractions: 100
    fractional_unit: Piastre
    polity: SSD
    monetary_authority: Bank of South Sudan
  STN:
    name: SaoTomePrincipeDobra
    label: São Tomé and Príncipe dobra
    symbol: "Db"
    fractions: 100
    fractional_unit: Cêntimo
    polity: STP
    monetary_authority: Central Bank of São Tomé and Príncipe
  SYP:
    name: SyrianPound
    label: Syrian pound
    symbol: "£"
    fractions: 100
    fractional_unit: Piastre
    polity: SYR
    monetary_authority: Central Bank of Syria
  SZL:
    name: SwaziLilangeni
    label: Swazi lilangeni
    symbol: "L"
    fractions: 100
    fractional_unit: Cent
    polity: SWZ
    monetary_authority: Central Bank of Swaziland
  THB:
    name: ThaiBaht
    label: Thai baht
    symbol: "฿"
    fractions: 100
    fractional_unit: Satang
    polity: THA
    monetary_authority: Bank of Thailand
  TJS:
    name: TajikistaniSomoni
    label: Tajikistani somoni
    symbol: "ЅМ"
    fractions: 100
    fractional_unit: Diram
    polity: TJK
    monetary_authority: National Bank of Tajikistan
  TMT:
    name: TurkmenistanManat
    label: Turkmenistan manat
    symbol: "m"
    fractions: 100
    fractional_unit: Tennesi
    polity: TKM
    monetary_authority: Central Bank of Turkmenistan
  TND:
    name: TunisianDinar
    label: Tunisian dinar
    symbol: "د.ت"
    fractions: 1000
    fractional_unit: Millime
    polity: TUN
    monetary_authority: Central Bank of Tunisia
  TOP:
    name: TonganPaAnga
    label: Tongan paʻanga
    symbol: "T$"
    fractions: 100
    fractional_unit: Seniti
    polity: TON
    monetary_authority: National Reserve Bank of Tonga
  TRY:
    name: TurkishLira
    label: Turkish lira
    symbol: "₺"
    fractions: 100
    fractional_unit: Kuruş
    polity: TUR
    monetary_authority: Central Bank of the Republic of Turkey
  TTD:
    name: TrinidadTobagoDollar
    label: Trinidad and Tobago dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: TTO
    monetary_authority: Central Bank of Trinidad and Tobago
  TWD:
    name: NewTaiwanDollar
    label: New Taiwan dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: TWN
    monetary_authority: Central Bank of the Republic of China (Taiwan)
  TZS:
    name: TanzanianShilling
    label: Tanzanian shilling
    symbol: "Sh"
    fractions: 100
    fractional_unit: Cent
    polity: TZA
    monetary_authority: Bank of Tanzania
  UAH:
    name: UkrainianHryvnia
    label: Ukrainian hryvnia
    symbol: "₴"
    fractions: 100
    fractional_unit: Kopiyka
    polity: UKR
    monetary_authority: National Bank of Ukraine
  UGX:
    name: UgandanShilling
    label: Ugandan shilling
    symbol: "Sh"
    fractions: 100
    fractional_unit: Cent
    polity: UGA
    monetary_authority: Bank of Uganda
  USD:
    name: UnitedStatesDollar
    label: United States dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity: USA, ASM, VGB, BES, ECU, SLV, GUM, HTI, MHL, FSM, MNP, PLW, PAN, PRI, TLS, TCA, VIR, UMI
    monetary_authority: Federal Reserve
  UYU:
    name: UruguayanPeso
    label: Uruguayan peso
    symbol: "$"
    fractions: 100
    fractional_unit: Centésimo
    polity: URY
    monetary_authority: Central Bank of Uruguay
  UYU:
    name: UruguayanPeso
    label: Uruguayan peso
    symbol: "$"
    fractions: 100
    fractional_unit: Centésimo
    polity: URY
    monetary_authority: Central Bank of Uruguay
  UZS:
    name: UzbekistaniSom
    label: Uzbekistani soʻm
    symbol: "so'm"
    fractions: 100
    fractional_unit: Tiyin
    polity: UZB
    monetary_authority: Central Bank of the Republic of Uzbekistan
  VES:
    name: VenezuelanBolivarSoberano
    label: Venezuelan bolívar soberano
    symbol: "Bs."
    fractions: 100
    fractional_unit: Céntimo
    polity: VEN
    monetary_authority: Central Bank of Venezuela
  VND:
    name: VietlabelseDong
    label: Vietlabelse đồng
    symbol: "₫"
    fractions: 100
    fractional_unit: Hào
    polity: VNM
    monetary_authority: State Bank of Vietnam
  VUV:
    name: VanuatuVatu
    label: Vanuatu vatu
    symbol: "Vt"
    fractions: 1
    fractional_unit: ~
    polity: VUT
    monetary_authority: Reserve Bank of Vanuatu
  WST:
    name: SamoanTala
    label: Samoan tālā
    symbol: "T"
    fractions: 100
    fractional_unit: Sene
    polity: WSM
    monetary_authority: Central Bank of Samoa
  XAF:
    name: CentralAfricanCFAFranc
    label: Central African CFA franc
    symbol: "T"
    fractions: 100
    fractional_unit: Sene
    polity: CMR, CAF, COG, TCD, GNQ, GAB
    monetary_authority: Central Bank of Samoa
#XAG	.	Silver (one troy ounce)
#XAU	.	Gold (one troy ounce)
#XBA	.	European Composite Unit (EURCO) (bond market unit)
#XBB	.	European Monetary Unit (E.M.U.-6) (bond market unit)
#XBC	.	European Unit of Account 9 (E.U.A.-9) (bond market unit)
#XBD	.	European Unit of Account 17 (E.U.A.-17) (bond market unit)
  XCD:
    name: EastCaribbeanDollar
    label: East Caribbean dollar
    symbol: "$"
    fractions: 100
    fractional_unit: Cent
    polity:	AIA, ATG, DMA, GRD, MSR, KNA, LCA, VCT
    monetary_authority: Eastern Caribbean Central Bank
#XDR	.	Special drawing rights	International Monetary Fund
  XOF:
    name: WestAfricanCFAFranc
    label: West African CFA franc
    symbol:	"Fr"
    fractions: 100
    fractional_unit: Centime
    polity: BEN, BFA, CIV, GNB, MLI, NER, SEN, TGO
    monetary_authority: Central Bank of West African States
#XPD	.	Palladium (one troy ounce)
  XPF:
#  	0	CFP franc (franc Pacifique)	French territories of the Pacific Ocean:  French Polynesia (PF),  NewCaledonia (NC),  Wallis and Futuna (WF)
#XPT	.	Platinum (one troy ounce)
  YER:
    name: YemeniRial
    label: Yemeni rial
    symbol: "﷼"
    fractions: 100
    fractional_unit: Fils
    polity: YEM
    monetary_authority: Central Bank of Yemen
  ZAR:
    name: SouthAfricanRand
    label: South African rand
    symbol: "R"
    fractions: 100
    fractional_unit: Cent
    polity: ZAF
    monetary_authority: South African Reserve Bank
  ZMW:
    name: ZambianKwacha
    label: Zambian kwacha
    symbol: "ZK"
    fractions: 100
    fractional_unit: Ngwee
    polity: ZMB
    monetary_authority: Bank of Zambia

`

// Entities - Legal Entities & Ownership Structures
var yamlEntities = `

metadata:
  name: Entities
  description: "Legal Entities & Ownership Structures"
  type: EntityType


values:
    I:
      name: individual # (Natural Person)
      label: "Individual"
      type: Legal
      roles:
        principal: null
        legalGuardian: []
        agent: [] # (granted by the principal)
        ancillary:
          accountant: []
          advisor: []
          lawyer: []
          manager: []
          trader: []
    P:
      name: publicCompany
      label: "Public Company Limited by Shares"
      type: Legal
      roles:
        board:
          chairman: null
          director: []
          secretary: null
        officers:
          ceo: null
          coo: null
          cfo: null
          cto: null
          executive: []
        ancillary:
          accountant: []
          advisor: []
          lawyer: []
          manager: []
          trader: []
        ownership:
          member: []
          significantMember: []
    C:
      name: privateCompany
      label: "Private Company Limited by Shares"
      type: Legal
      roles:
        board:
          chairman: null
          director: []
          secretary: null
        officers:
          ceo: null
          coo: null
          cfo: null
          cto: null
          executive: []
        ancillary:
          accountant: []
          advisor: []
          lawyer: []
          manager: []
          trader: []
        ownership:
          member: []
          significantMember: []
    L:
      name: limitedPartnership
      label: "Limited Partnership"
      type: Legal
      roles:
        managingPartner: []
        partner: []
        ancillary:
          accountant: []
          advisor: []
          lawyer: []
          manager: []
          trader: []
    U:
      name: unlimitedPartnership
      label: "Unlimited Partnership" # (General Partnership, Marriage, Civil Union, Common Law Marriage, Domestic Partnership,  )
      type: Legal
      roles:
        managingPartner: []
        partner: []
        ancillary:
          accountant: []
          advisor: []
          lawyer: []
          manager: []
          trader: []
    T:
      name: soleProprietorship
      label: "Sole Proprietorship"
      type: Legal
      roles:
        proprietor: null
        agent: [] # (granted by the proprietor)
        ancillary:
          accountant: []
          advisor: []
          lawyer: []
          manager: []
          trader: []
    S:
      name: statutoryCompany
      label: "Statutory Company"
      type: Legal
      roles:
        board:
          chairman: null
          director: []
          secretary: null
        officers:
          ceo: null
          coo: null
          cfo: null
          cto: null
          executive: []
        ancillary:
          accountant: []
          advisor: []
          lawyer: []
          manager: []
          trader: []
    O:
      name: nonProfitOrganization
      label: "Non-Profit Organization"
      type: Legal
      role:
        board:
          chairman: null
          director: []
          secretary: null
        officers:
          ceo: null
          coo: null
          cfo: null
          cto: null
          executive: []
        ancillary:
          accountant: []
          advisor: []
          lawyer: []
          manager: []
          trader: []
    N:
      name: nationState
      label: "Nation State" # (Sovereign State)
      type: Legal
    G:
      name: governmentAgency
      label: "Government Agency"
      type: Legal
    U:
      name: unitTrust
      label: "Unit Trust"
      type: Ownership
      roles:
        trustee: []
        beneficiary: []
        settlor: []
        protector: []
        ancillary:
          accountant: []
          advisor: []
          custodian: []
          lawyer: []
          manager: []
          trader: []
    D:
      name: discretionaryTrust
      label: "Discretionary Trust" # (Family Trust)
      type: Ownership
      roles:
        trustee: []
        beneficiary: []
        settlor: []
        protector: []
        ancillary:
          accountant: []
          advisor: []
          custodian: []
          lawyer: []
          manager: []
          trader: []

`

// Messages - Message types. Used to categorize messages and assign special
// meaning.
var yamlMessages = `

metadata:
  name: Messages
  description: "Message types. Used to categorize messages and assign special meaning."
  type: MessageType


values:

    # Public message, public announcement. Sent to another PKH or back to the same PKH.  Can be used for an official issuer annoucement.
    2:
      name: Public Message (Generic)

    # Generic private message. DH, RSA, etc.
    3:
      name: Private Message (Generic)

    # A message that is meant to signal intent or specify an offer to another party (PKH).  The offer can be for a token, or for a static contract.  An offer is effectively a message with no signatures on it, and may have non-standardized communication.  "I'd like to offer you 1000 BCH for your shares in Apple."  It can be the start of a negotiation.  Likely done in a private manner after establishing a secure (DH) communication channel.
    1001:
      name: Offer

    # A partially signed transaction or a Payment Request.  Useful for sending transactions that require multiple signatures from different contracting parties such as in the case of bitcoin multisig txns and exchange (token) actions. Also useful for static contract formation.  The Payment Request can be sent along with an Invoice.  The Invoicee can use the Payment Request to actually make the payment.
    1002:
      name: Signature Request

    ###############################################################################################################################
    #                           Message Codes below this are still in flux and should not be used.                                #
    ###############################################################################################################################

    # The Owner of the address declares the rules (message sent back to itself) by which they are willing to accept messages. Certain types of messages can be excluded, and most importantly the minimum threshold value that must be sent in order to avoid being considered spam.  Different threshold amounts can be applied to different types of messages.  The Public Key (RSA) can also be declared here.
    0:
      name: Communication Rules

    # Business Name, Registration Number, Address, Communication Details (Phone/Email/etc.), Key People
    1:
      name: Commercial Profile

    # YA is Contract Offered to a PKH that the sender is looking to establish a secure line of communication with. Useful for establishing a secure communication channel for a business-to-business relationship such as a supplier-customer relationship.
    4:
      name: Establish Private Messaging (DH) YA (1 of 2)

    # YB is Contract Offered in response to a YA message.  Both parties can now create encrypted messages that they can both read. Useful for establishing a secure communication channel for a business-to-business relationship such as a supplier-customer relationship.
    5:
      name: Establish Private Messaging (DH) YB (2 of 2)

    # RSA public key or the like.  This message is sent back to one's self to publicly broadcast the public encryption key associated with the PKH.  This allows for permissionless and anonymous 1-way communication with the owner of the public key.
    6:
      name: Public Key

    # The sender sends the PKH that they want to receive all of the messages/payments to.
    7:
      name: Trading Address (PKH)

    # Either sent back to yourself, or accompanying a txn. Often encrypted (DH, RSA) to make a note/receipt against a particular output. DH for b2b where both parties can see the receipt/note.  RSA or the like for receipts only visible to one of the transacting parties.
    8:
      name: Receipt/Output Note

    # A quote.  Can be used to generate a Purchase Order.
    9:
      name: Quote

    # Sent by the customer to the supplier.  Often generated in response to a quote. Can be securitized with a Payable token. (Factoring/Reverse-Factoring)
    10:
      name: Purchase Order

    # An official acknowledgement of a purchase order.  May include information such as ETA or material shortfalls.
    11:
      name: Purchase Order Acknowledgment

    # Also known as a packing slip.  Sent to customers as an advance notice of what has been shipped.
    12:
      name: Advance Ship Notice

    # An invoice. Invoices can be securitized by creating a receivable token.
    13:
      name: Invoice

    # An official list of products offered by a supplier.  Can include pricing as well.
    14:
      name: Product Catalog

    # A forecast notification from the manufacturer to the relevant supplier of the qty of materials needed and the time period in which they are needed.
    15:
      name: Planning Schedule/Material Release

    # An authorization from the manufacturer to the supplier to ship goods according to a specific short-term schedule. It provides detailed shipping requirements and adds more specific instruction to the Planning Schedule/Material Release that may have been provided earlier.
    16:
      name: Shipping Schedule

    #
    17:
      name: Collateral Call

`

// Polities - Polities (eg. Countries/Nation-States (ISO-3166 Alpha-3),
// Political Unions, International Organizations, etc.)
var yamlPolities = `

metadata:
  name: Polities
  description: "Polities (eg. Countries/Nation-States (ISO-3166 Alpha-3), Political Unions, International Organizations, etc.)"
  type: Polity


values:
  ALA:
    name: Aaland Islands
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/5/52/Flag_of_%C3%85land.svg"
  AFG:
    name: Afghanistan
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/9/9a/Flag_of_Afghanistan.svg"
  ALB:
    name: Albania
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/3/36/Flag_of_Albania.svg"
  DZA:
    name: Algeria
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/7/77/Flag_of_Algeria.svg"
  ASM:
    name: American Samoa
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/8/87/Flag_of_American_Samoa.svg"
  AND:
    name: Andorra
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/1/19/Flag_of_Andorra.svg"
  AGO:
    name: Angola
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/9/9d/Flag_of_Angola.svg"
  AIA:
    name: Anguilla
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/b/b4/Flag_of_Anguilla.svg"
  ATA:
    name: Antarctica
    states: ~
    flag:
  ATG:
    name: Antigua and Barbuda
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/8/89/Flag_of_Antigua_and_Barbuda.svg"
  ARG:
    name: Argentina
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/1/1a/Flag_of_Argentina.svg"
  ARM:
    name: Armenia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/2/2f/Flag_of_Armenia.svg"
  ABW:
    name: Aruba
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/f/f6/Flag_of_Aruba.svg"
  AU:
    name: African Union
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
  AUS:
    name: Australia
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
  AUT:
    name: Austria
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/4/41/Flag_of_Austria.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231
  AZE:
    name: Azerbaijan
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/d/dd/Flag_of_Azerbaijan.svg"
  BHS:
    name: The Bahamas
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/9/93/Flag_of_the_Bahamas.svg"
  BHR:
    name: Bahrain
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/2/2c/Flag_of_Bahrain.svg"
  BGD:
    name: Bangladesh
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/f/f9/Flag_of_Bangladesh.svg"
    fiscal_year: 0701-0630
    gov_fiscal_year: 0701-0630
  BRB:
    name: Barbados
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/e/ef/Flag_of_Barbados.svg"
  BLR:
    name: Belarus
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/8/85/Flag_of_Belarus.svg"
  BEL:
    name: Belgium
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/6/65/Flag_of_Belgium.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231
  BLZ:
    name: Belize
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/e/e7/Flag_of_Belize.svg"
  BEN:
    name: Benin
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/0a/Flag_of_Benin.svg"
  BMU:
    name: Bermuda
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/b/bf/Flag_of_Bermuda.svg"
  BTN:
    name: Bhutan
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/9/91/Flag_of_Bhutan.svg"
  BOL:
    name: Bolivia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/d/de/Flag_of_Bolivia_%28state%29.svg"
  BES:
    name: Bonaire, St Eustasuis and Saba
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/2/20/Flag_of_the_Netherlands.svg"
  BIH:
    name: Bosnia and Herzegovina
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/b/bf/Flag_of_Bosnia_and_Herzegovina.svg"
  BWA:
    name: Botswana
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/f/fa/Flag_of_Botswana.svg"
  BVT:
    name: Bouvet Island
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/d/d9/Flag_of_Norway.svg"
  BRA:
    name: Brazil
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
  IOT:
    name: British Indian Ocean Territory
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/6/6e/Flag_of_the_British_Indian_Ocean_Territory.svg"
  VGB:
    name: British Virgin Islands
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/4/42/Flag_of_the_British_Virgin_Islands.svg"
  BRN:
    name: Brunei
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/9/9c/Flag_of_Brunei.svg"
  BGR:
    name: Bulgaria
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/9/9a/Flag_of_Bulgaria.svg"
  BFA:
    name: Burkina Faso
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/3/31/Flag_of_Burkina_Faso.svg"
  BDI:
    name: Burundi
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/5/50/Flag_of_Burundi.svg"
  KHM:
    name: Cambodia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/8/83/Flag_of_Cambodia.svg"
  CMR:
    name: Cameroon
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/4/4f/Flag_of_Cameroon.svg"
  CAN:
    name: Canada
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
  CPV:
    name: Cape Verde
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/3/38/Flag_of_Cape_Verde.svg"
  CYM:
    name: Cayman Islands
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/0f/Flag_of_the_Cayman_Islands.svg"
  CAF:
    name: Central African Republic
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/6/6f/Flag_of_the_Central_African_Republic.svg"
  TCD:
    name: Chad
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/4/4b/Flag_of_Chad.svg"
  CHL:
    name: Chile
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/7/78/Flag_of_Chile.svg"
  CHN:
    name: China
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
  CXR:
    name: Christmas Island
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/6/67/Flag_of_Christmas_Island.svg"
  CCK:
    name: Cocos Islands
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/7/74/Flag_of_the_Cocos_%28Keeling%29_Islands.svg"
  COL:
    name: Colombia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/2/21/Flag_of_Colombia.svg"
  COM:
    name: Comoros
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/9/94/Flag_of_the_Comoros.svg"
  COG:
    name: Congo-Brazzaville
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/9/92/Flag_of_the_Republic_of_the_Congo.svg"
  COD:
    name: Congo-Kinshasa
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/6/6f/Flag_of_the_Democratic_Republic_of_the_Congo.svg"
  COK:
    name: Cook Islands
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/3/35/Flag_of_the_Cook_Islands.svg"
  CRI:
    name: Costa Rica
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/b/bc/Flag_of_Costa_Rica_%28state%29.svg"
    fiscal_year: 1001-0931
    gov_fiscal_year: 1001-0931
  HRV:
    name: Croatia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/1/1b/Flag_of_Croatia.svg"
  CUB:
    name: Cuba
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/b/bd/Flag_of_Cuba.svg"
  CUW:
    name: Curacao
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/b/b1/Flag_of_Cura%C3%A7ao.svg"
  CYP:
    name: Cyprus
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/d/d4/Flag_of_Cyprus.svg"
  CZE:
    name: Czech Republic
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/c/cb/Flag_of_the_Czech_Republic.svg"
  DNK:
    name: Denmark
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/9/9c/Flag_of_Denmark.svg"
  DJI:
    name: Djibouti
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/3/34/Flag_of_Djibouti.svg"
  DMA:
    name: Dominica
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/c/c4/Flag_of_Dominica.svg"
  DOM:
    name: Dominican Republic
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/9/9f/Flag_of_the_Dominican_Republic.svg"
  TLS:
    name: East Timor
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/2/26/Flag_of_East_Timor.svg"
  ECU:
    name: Ecuador
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/e/e8/Flag_of_Ecuador.svg"
  EGY:
    name: Egypt
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/f/fe/Flag_of_Egypt.svg"
    fiscal_year: 0701-0630
    gov_fiscal_year: 0701-0630
  SLV:
    name: El Salvador
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/3/34/Flag_of_El_Salvador.svg"
  GNQ:
    name: Equatorial Guinea
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/3/31/Flag_of_Equatorial_Guinea.svg"
  ERI:
    name: Eritrea
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/2/29/Flag_of_Eritrea.svg"
  EST:
    name: Estonia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/8/8f/Flag_of_Estonia.svg"
  ETH:
    name: Ethiopia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/7/71/Flag_of_Ethiopia.svg"
    fiscal_year: 0708-0707
    gov_fiscal_year: 0708-0707
  EU:
    name: European Union
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
  FLK:
    name: Falkland Islands
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/8/83/Flag_of_the_Falkland_Islands.svg"
  FRO:
    name: Faroe Islands
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/3/3c/Flag_of_the_Faroe_Islands.svg"
  FJI:
    name: Fiji
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/b/ba/Flag_of_Fiji.svg"
  FIN:
    name: Finland
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/b/bc/Flag_of_Finland.svg"
  FRA:
    name: France
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/en/c/c3/Flag_of_France.svg"
  GUF:
    name: French Guiana
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/en/c/c3/Flag_of_France.svg"
  PYF:
    name: French Polynesia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/d/db/Flag_of_French_Polynesia.svg"
  ATF:
    name: French Southern and Antarctic Lands
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/a/a7/Flag_of_the_French_Southern_and_Antarctic_Lands.svg"
  GAB:
    name: Gabon
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/04/Flag_of_Gabon.svg"
  GMB:
    name: The Gambia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/7/77/Flag_of_The_Gambia.svg"
  GEO:
    name: Georgia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/0f/Flag_of_Georgia.svg"
  DEU:
    name: Germany
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/en/b/ba/Flag_of_Germany.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231
  GHA:
    name: Ghana
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/1/19/Flag_of_Ghana.svg"
  GIB:
    name: Gibraltar
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/02/Flag_of_Gibraltar.svg"
  GRC:
    name: Greece
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/5/5c/Flag_of_Greece.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231
  GRL:
    name: Greenland
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/09/Flag_of_Greenland.svg"
  GRD:
    name: Grenada
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/b/bc/Flag_of_Grenada.svg"
  GLP:
    name: Guadeloupe
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/en/c/c3/Flag_of_France.svg"
  GUM:
    name: Guam
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/07/Flag_of_Guam.svg"
  GTM:
    name: Guatemala
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/e/ec/Flag_of_Guatemala.svg"
  GGY:
    name: Guernsey
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/f/fa/Flag_of_Guernsey.svg"
  GIN:
    name: Guinea
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/e/ed/Flag_of_Guinea.svg"
  GNB:
    name: Guinea-Bissau
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/01/Flag_of_Guinea-Bissau.svg"
  GUY:
    name: Guyana
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/9/99/Flag_of_Guyana.svg"
  HTI:
    name: Haiti
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/5/56/Flag_of_Haiti.svg"
  HMD:
    name: Heard Island and McDonald Islands
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/8/88/Flag_of_Australia_%28converted%29.svg"
  HND:
    name: Honduras
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/8/82/Flag_of_Honduras.svg"
  HKG:
    name: Hong Kong
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/5/5b/Flag_of_Hong_Kong.svg"
    fiscal_year: 0401-0331
    gov_fiscal_year: 0401-0331
  HUN:
    name: Hungary
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/c/c1/Flag_of_Hungary.svg"
  ISL:
    name: Iceland
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/c/ce/Flag_of_Iceland.svg"
  IND:
    name: India
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
  IDN:
    name: Indonesia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/9/9f/Flag_of_Indonesia.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231
  IRN:
    name: Iran
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/c/ca/Flag_of_Iran.svg"
  IRQ:
    name: Iraq
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/f/f6/Flag_of_Iraq.svg"
  IRL:
    name: Ireland
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/4/45/Flag_of_Ireland.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231
  IMN:
    name: Isle of Man
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/5/5d/Flag_of_the_Isle_of_Mann.svg"
  ISR:
    name: Israel
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/d/d4/Flag_of_Israel.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231
  ITA:
    name: Italy
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/en/0/03/Flag_of_Italy.svg"
  CIV:
    name: Ivory Coast
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/f/fe/Flag_of_C%C3%B4te_d%27Ivoire.svg"
  JAM:
    name: Jamaica
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/0a/Flag_of_Jamaica.svg"
  JPN:
    name: Japan
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/en/9/9e/Flag_of_Japan.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0401-0331
  JEY:
    name: Jersey
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/1/1c/Flag_of_Jersey.svg"
  JOR:
    name: Jordan
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/c/c0/Flag_of_Jordan.svg"
  KAZ:
    name: Kazakhstan
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/d/d3/Flag_of_Kazakhstan.svg"
  KEN:
    name: Kenya
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/4/49/Flag_of_Kenya.svg"
  KIR:
    name: Kiribati
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/d/d3/Flag_of_Kiribati.svg"
  KWT:
    name: Kuwait
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/a/aa/Flag_of_Kuwait.svg"
  KGZ:
    name: Kyrgyzstan
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/c/c7/Flag_of_Kyrgyzstan.svg"
  LAO:
    name: Laos
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/5/56/Flag_of_Laos.svg"
  LVA:
    name: Latvia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/8/84/Flag_of_Latvia.svg"
  LBN:
    name: Lebanon
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/5/59/Flag_of_Lebanon.svg"
  LSO:
    name: Lesotho
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/4/4a/Flag_of_Lesotho.svg"
  LBR:
    name: Liberia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/b/b8/Flag_of_Liberia.svg"
  LBY:
    name: Libya
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/05/Flag_of_Libya.svg"
  LIE:
    name: Liechtenstein
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/4/47/Flag_of_Liechtenstein.svg"
  LTU:
    name: Lithuania
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/1/11/Flag_of_Lithuania.svg"
  LUX:
    name: Luxembourg
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/d/da/Flag_of_Luxembourg.svg"
  MAC:
    name: Macau
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/6/63/Flag_of_Macau.svg"
  MKD:
    name: Macedonia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/f/f8/Flag_of_Macedonia.svg"
  MDG:
    name: Madagascar
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/b/bc/Flag_of_Madagascar.svg"
  MWI:
    name: Malawi
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/d/d1/Flag_of_Malawi.svg"
  MYS:
    name: Malaysia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/6/66/Flag_of_Malaysia.svg"
  MDV:
    name: Maldives
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/0f/Flag_of_Maldives.svg"
  MLI:
    name: Mali
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/9/92/Flag_of_Mali.svg"
  MLT:
    name: Malta
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/7/73/Flag_of_Malta.svg"
  MHL:
    name: Marshall Islands
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/2/2e/Flag_of_the_Marshall_Islands.svg"
  MTQ:
    name: Martinique
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/en/c/c3/Flag_of_France.svg"
  MRT:
    name: Mauritania
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/3/30/New_National_Flag_of_Mauritania.png"
  MUS:
    name: Mauritius
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/7/77/Flag_of_Mauritius.svg"
  MYT:
    name: Mayotte
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/4/4a/Flag_of_Mayotte_%28local%29.svg"
  MEX:
    name: Mexico
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
  FSM:
    name: Micronesia
    states: ~
    flag:
  MDA:
    name: Moldova
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/2/27/Flag_of_Moldova.svg"
  MCO:
    name: Monaco
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/e/ea/Flag_of_Monaco.svg"
  MNG:
    name: Mongolia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/4/4c/Flag_of_Mongolia.svg"
  MNE:
    name: Montenegro
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/6/64/Flag_of_Montenegro.svg"
  MSR:
    name: Montserrat
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/d/d0/Flag_of_Montserrat.svg"
  MAR:
    name: Morocco
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/2/2c/Flag_of_Morocco.svg"
  MOZ:
    name: Mozambique
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/d/d0/Flag_of_Mozambique.svg"
  MMR:
    name: Myanmar
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/8/8c/Flag_of_Myanmar.svg"
  NAM:
    name: Namibia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/00/Flag_of_Namibia.svg"
  NRU:
    name: Nauru
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/3/30/Flag_of_Nauru.svg"
  NPL:
    name: Nepal
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/9/9b/Flag_of_Nepal.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231
  NLD:
    name: Netherlands
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/2/20/Flag_of_the_Netherlands.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231
  NCL:
    name: New Caledonia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/en/c/c3/Flag_of_France.svg"
  NZL:
    name: New Zealand
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/3/3e/Flag_of_New_Zealand.svg"
  NIC:
    name: Nicaragua
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/1/19/Flag_of_Nicaragua.svg"
  NER:
    name: Niger
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/f/f4/Flag_of_Niger.svg"
  NGA:
    name: Nigeria
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/7/79/Flag_of_Nigeria.svg"
  NIU:
    name: Niue
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/01/Flag_of_Niue.svg"
  NFK:
    name: Norfolk and Philip Island
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/4/48/Flag_of_Norfolk_Island.svg"
  PRK:
    name: North Korea
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/5/51/Flag_of_North_Korea.svg"
  MNP:
    name: Northern Mariana Islands
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/e/e0/Flag_of_the_Northern_Mariana_Islands.svg"
  NOR:
    name: Norway
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/d/d9/Flag_of_Norway.svg"
  OMN:
    name: Oman
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/d/dd/Flag_of_Oman.svg"
  PAK:
    name: Pakistan
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/3/32/Flag_of_Pakistan.svg"
  PLW:
    name: Palau
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/4/48/Flag_of_Palau.svg"
  PSE:
    name: Palestinian Territory
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/00/Flag_of_Palestine.svg"
  PAN:
    name: Panama
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/a/ab/Flag_of_Panama.svg"
  PNG:
    name: Papua New Guinea
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/e/e3/Flag_of_Papua_New_Guinea.svg"
  PRY:
    name: Paraguay
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/2/27/Flag_of_Paraguay.svg"
  PER:
    name: Peru
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/d/df/Flag_of_Peru_%28state%29.svg"
  PHL:
    name: Philippines
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/9/99/Flag_of_the_Philippines.svg"
  PCN:
    name: Pitcairn Islands
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/8/88/Flag_of_the_Pitcairn_Islands.svg"
  POL:
    name: Poland
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/en/1/12/Flag_of_Poland.svg"
  PRT:
    name: Portugal
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/5/5c/Flag_of_Portugal.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231
  PRI:
    name: Puerto Rico
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/2/28/Flag_of_Puerto_Rico.svg"
  QAT:
    name: Qatar
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/6/65/Flag_of_Qatar.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231
  REU:
    name: Réunion
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/5/5a/Flag_of_R%C3%A9union.svg"
  ROU:
    name: Romania
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/7/73/Flag_of_Romania.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231
  RUS:
    name: Russia
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
  RWA:
    name: Rwanda
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/1/17/Flag_of_Rwanda.svg"
  SHN:
    name: Saint Helena, Ascension and Tristan da Cunha
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/en/a/ae/Flag_of_the_United_Kingdom.svg"
  KNA:
    name: Saint Kitts and Nevis
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/f/fe/Flag_of_Saint_Kitts_and_Nevis.svg"
  LCA:
    name: Saint Lucia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/9/9f/Flag_of_Saint_Lucia.svg"
  SPM:
    name: Saint Pierre and Miquelon
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/en/c/c3/Flag_of_France.svg"
  VCT:
    name: Saint Vincent and the Grenadines
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/6/6d/Flag_of_Saint_Vincent_and_the_Grenadines.svg"
  BLM:
    name: Saint-Barthelemy
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/d/df/Flag_of_Saint_Barthelemy_%28local%29.svg"
  MAF:
    name: Saint-Martin
    states: ~
    flag:
  WSM:
    name: Samoa
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/3/31/Flag_of_Samoa.svg"
  SMR:
    name: San Marino
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/b/b1/Flag_of_San_Marino.svg"
  STP:
    name: São Tomé and Príncipe
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/4/4f/Flag_of_Sao_Tome_and_Principe.svg"
  SAU:
    name: Saudi Arabia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/0d/Flag_of_Saudi_Arabia.svg"
  SEN:
    name: Senegal
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/f/fd/Flag_of_Senegal.svg"
  SRB:
    name: Serbia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/f/ff/Flag_of_Serbia.svg"
  SYC:
    name: Seychelles
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/f/fc/Flag_of_Seychelles.svg"
  SLE:
    name: Sierra Leone
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/1/17/Flag_of_Sierra_Leone.svg"
  SGP:
    name: Singapore
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/4/48/Flag_of_Singapore.svg"
  SXM:
    name: Sint Maarten
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/d/d3/Flag_of_Sint_Maarten.svg"
  SVK:
    name: Slovakia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/e/e6/Flag_of_Slovakia.svg"
  SVN:
    name: Slovenia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/f/f0/Flag_of_Slovenia.svg"
  SLB:
    name: Solomon Islands
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/7/74/Flag_of_the_Solomon_Islands.svg"
  SOM:
    name: Somalia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/a/a0/Flag_of_Somalia.svg"
  ZAF:
    name: South Africa
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/a/af/Flag_of_South_Africa.svg"
  SGS:
    name: South Georgia and the South Sandwich Islands
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/e/ed/Flag_of_South_Georgia_and_the_South_Sandwich_Islands.svg"
  KOR:
    name: South Korea
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/09/Flag_of_South_Korea.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231
  SSD:
    name: South Sudan
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/7/7a/Flag_of_South_Sudan.svg"
  ESP:
    name: Spain
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/en/9/9a/Flag_of_Spain.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231
  LKA:
    name: Sri Lanka
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/1/11/Flag_of_Sri_Lanka.svg"
  SDN:
    name: Sudan
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/01/Flag_of_Sudan.svg"
  SUR:
    name: Suriname
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/6/60/Flag_of_Suriname.svg"
  SJM:
    name: Svalbard and Jan Mayen
    states: ~
    flag:
  SWZ:
    name: Swaziland
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/f/fb/Flag_of_Eswatini.svg"
  SWE:
    name: Sweden
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/en/4/4c/Flag_of_Sweden.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231
  CHE:
    name: Switzerland
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/08/Flag_of_Switzerland_%28Pantone%29.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231
  SYR:
    name: Syria
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/5/53/Flag_of_Syria.svg"
  TWN:
    name: Taiwan
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/7/72/Flag_of_the_Republic_of_China.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231
  TJK:
    name: Tajikistan
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/d/d0/Flag_of_Tajikistan.svg"
  TZA:
    name: Tanzania
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/3/38/Flag_of_Tanzania.svg"
  THA:
    name: Thailand
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/a/a9/Flag_of_Thailand.svg"
  TGO:
    name: Togo
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/6/68/Flag_of_Togo.svg"
  TKL:
    name: Tokelau
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/8/8e/Flag_of_Tokelau.svg"
  TON:
    name: Tonga
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/9/9a/Flag_of_Tonga.svg"
  TTO:
    name: Trinidad and Tobago
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/6/64/Flag_of_Trinidad_and_Tobago.svg"
  TUN:
    name: Tunisia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/c/ce/Flag_of_Tunisia.svg"
  TUR:
    name: Turkey
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/b/b4/Flag_of_Turkey.svg"
  TKM:
    name: Turkmenistan
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/1/1b/Flag_of_Turkmenistan.svg"
  TCA:
    name: Turks and Caicos Islands
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/a/a0/Flag_of_the_Turks_and_Caicos_Islands.svg"
  TUV:
    name: Tuvalu
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/3/38/Flag_of_Tuvalu.svg"
  UGA:
    name: Uganda
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/4/4e/Flag_of_Uganda.svg"
  UKR:
    name: Ukraine
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/4/49/Flag_of_Ukraine.svg"
  ARE:
    name: United Arab Emirates
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/c/cb/Flag_of_the_United_Arab_Emirates.svg"
    fiscal_year: 0101-1231
    gov_fiscal_year: 0101-1231
  GBR:
    name: United Kingdom
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/en/a/ae/Flag_of_the_United_Kingdom.svg"
    fiscal_year: 0406-0405
    gov_fiscal_year: 0401-0331
  UMI:
    name: United States Minor Outlying Islands
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/en/a/a4/Flag_of_the_United_States.svg"
  URY:
    name: Uruguay
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/f/fe/Flag_of_Uruguay.svg"
  VIR:
    name: US Virgin Islands
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/f/f8/Flag_of_the_United_States_Virgin_Islands.svg"
  USA:
    name: USA
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
  UZB:
    name: Uzbekistan
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/8/84/Flag_of_Uzbekistan.svg"
  VUT:
    name: Vanuatu
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/6/6e/Flag_of_Vanuatu_%28official%29.svg"
  VAT:
    name: Vatican City
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/00/Flag_of_the_Vatican_City.svg"
  VEN:
    name: Venezuela
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/7/7b/Flag_of_Venezuela_%28state%29.svg"
  VNM:
    name: Vietnam
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/2/21/Flag_of_Vietnam.svg"
  WLF:
    name: Wallis and Futuna
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/en/c/c3/Flag_of_France.svg"
  ESH:
    name: Western Sahara
    states: ~
    flag:
  YEM:
    name: Yemen
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/8/89/Flag_of_Yemen.svg"
  ZMB:
    name: Zambia
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/0/06/Flag_of_Zambia.svg"
  ZWE:
    name: Zimbabwe
    states: ~
    flag: "https://upload.wikimedia.org/wikipedia/commons/6/6a/Flag_of_Zimbabwe.svg"

`

// RejectionCodes - Code/Text combinations returned in rejection messages
// when a request is not accepted.
var yamlRejectionCodes = `

metadata:
  name: RejectionCodes
  description: "Code/Text combinations returned in rejection messages when a request is not accepted."
  type: RejectionCode

values:
    #################################### Basic ####################################

    0:
      name: Success
      text: Success
      description: No failure. This code should not be used in a reject message.

    1:
      name: MsgMalformed
      text: Message Malformed
      description: The OP_RETURN message is malformed. It doesn't pass data validation or something about it isn't according to protocol.

    2:
      name: TxMalformed
      text: Transaction Malformed
      description: The Bitcoin tx is malformed. Incorrect inputs/outputs or something similar.

    #################################### Contract ####################################

    10:
      name: ContractExists
      text: Contract Already Exists
      description: The contract already exists and can't be recreated.

    11:
      name: ContractNotDynamic
      text: Contract Is Not Dynamic
      description: 'Sent when a CO is received, but the contract type is not "D" (Dynamic)'

    12:
      name: ContractAssetQtyReduction
      text: Contract Asset Quantity Reduction
      description: Sent when a CA tries to reduce the number of assets below the number of assets the contract has.

    13:
      name: ContractFixedQuantity
      text: Contract Fixed Quantity
      description: Sent when the issuer attempted to increase the quantity of assets in a contract beyond the fixed quantity permitted.

    14:
      name: ContractAuthFlags
      text: Contract Auth Flags Prohibit
      description: The contract auth flags don't permit the action requested.

    15:
      name: ContractExpired
      text: Contract Expired
      description: The contract is expired so can no longer accept requests.

    16:
      name: ContractFrozen
      text: Contract Frozen
      description: The contract is frozen and the request is not permitted while frozen.

    17:
      name: ContractRevision
      text: Contract Revision Incorrect
      description: The revision in a contract amendment is incorrect.

    #################################### Asset ####################################

    20:
      name: AssetCodeExists
      text: Asset Code Already Exists
      description: The asset code specified already exists and can't be reused.

    21:
      name: AssetNotFound
      text: Asset Not Found
      description: The asset code is not found.

    22:
      name: AssetAuthFlags
      text: Asset Auth Flags Prohibit
      description: The asset auth flags don't permit the action requested.

    23:
      name: AssetFrozen
      text: Asset Frozen
      description: The asset is frozen and the request is not permitted while frozen.

    24:
      name: AssetRevision
      text: Asset Revision Incorrect
      description: The revision in an asset amendment is incorrect.

    #################################### Transfer ####################################

    30:
      name: TransferSelf
      text: Transfer To Self Prohibited
      description: Transfers with the sender and receiver addresses the same are not permitted.

    31:
      name: TransferExpired
      text: Transfer Expired
      description: The transfer has expired.

    32:
      name: HoldingsFrozen
      text: Holdings Frozen
      description: Holdings are frozen, so the request can't be completed.

    #################################### Governance ####################################

    40:
      name: HolderProposalProhibited
      text: Holder Proposal Prohibited
      description: Holders are not permitted to make proposals.

    41:
      name: ProposalConflicts
      text: Proposal Conflicts
      description: The proposal conflicts with an unapplied proposal.

    42:
      name: VoteNotFound
      text: Vote Not Found
      description: The vote ID referenced is not found.

    43:
      name: VoteClosed
      text: Vote Closed
      description: The vote has closed and ballots are no longer permitted.

    44:
      name: BallotAlreadyCounted
      text: Ballot Already Counted
      description: The ballot has already been counted for this address.

    #################################### Enforcement ####################################

    #50:

    #################################### Funding ####################################

    60:
      name: InsufficientTxFeeFunding
      text: Insufficient Transaction Fee Funding
      description: Insufficient bitcoin quantities for response transaction fees.

    61:
      name: InsufficientValue
      text: Insufficient Value
      description: Insufficient bitcoin quantity in inputs to fund request.

    62:
      name: InsufficientQuantity
      text: Insufficient Quantity
      description: Insufficient token holdings to for request.

    #################################### Address ####################################

    70:
      name: NotIssuer
      text: Requestor Is Not Issuer
      description: The requestor is not the issuer and is required to be for this request.

    71:
      name: NotOperator
      text: Requestor Is Not Operator
      description: The requestor is not the operator and is required to be for this request.

    72:
      name: UnauthorizedAddress
      text: Unauthorized Address
      description: The address specified is not permitted for this request.

    #################################### Signatures ####################################

    80:
      name: InvalidSignature
      text: Invalid Signature
      description: The signature provided is not valid. This is for signatures included within OP_RETURN data. Not bitcoin transaction signature scripts.

`

// Roles - Roles that entities play in relation to their interactions with
// other entities. These roles have widely-accepted tasks, rights and
// duties.
var yamlRoles = `

metadata:
  name: Roles
  description: "Roles that entities play in relation to their interactions with other entities.  These roles have widely-accepted tasks, rights and duties."
  type: RoleType

values:
    1:
      name: Accountant
    2:
      name: Advisor
    3:
      name: Agent
    4:
      name: Beneficiary
    5:
      name: CEO
    6:
      name: CFO
    7:
      name: Chair
    8:
      name: COO
    9:
      name: CTO
    10:
      name: Custodian
    11:
      name: Director
    12:
      name: Executive
    13:
      name: Lawyer
    14:
      name: Legal Guardian
    15:
      name: Limited Partner
    16:
      name: Manager
    17:
      name: Managing Partner
    18:
      name: Member
      description: Shareholder
    19:
      name: Partner
    20:
      name: Principal
    21:
      name: Proprietor
    22:
      name: Protector
    23:
      name: Secretary
    24:
      name: Settlor
    25:
      name: Significant Member
      description: Major Shareholder
    26:
      name: Smart Contract Operator
    27:
      name: Trader
    28:
      name: Trustee



`
