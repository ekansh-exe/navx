-- 30 fixed SYSTEM_COMPANY cards (§5.1) + the NAV5 INDEX card (§5.2).
-- All names/symbols are invented, no real companies or brands.
-- Prices/supply are storage-unit integers (1 currency = 100 smallest units).

INSERT INTO cards (card_type, sector, symbol, name, supply_model, total_supply, circulating_supply, current_price, status) VALUES
    -- Food/Agriculture (6)
    ('SYSTEM_COMPANY', 'AGRICULTURE', 'GRNV', 'Granovale Grain Co.',        'FIXED', 2500000, 1200000, 2200, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'AGRICULTURE', 'PRDL', 'Pradero Livestock',          'FIXED', 2000000,  900000, 1800, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'FOOD',        'CACM', 'Cacoma Coffee & Cocoa',      'FIXED', 1500000,  600000, 3000, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'FOOD',        'TDVL', 'Tidevale Fisheries',         'FIXED', 1600000,  700000, 1400, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'FOOD',        'HSTD', 'Homestead Foods',            'FIXED', 2200000, 1000000, 2600, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'AGRICULTURE', 'FURW', 'Furrowtech Equipment',       'FIXED',  900000,  350000, 4000, 'ACTIVE'),
    -- Oil & Gas (6)
    ('SYSTEM_COMPANY', 'OIL_GAS', 'DPCR', 'Deepcrest Drilling',             'FIXED', 3000000, 1500000, 3500, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'OIL_GAS', 'BFLM', 'Brightflame Refining',           'FIXED', 2800000, 1300000, 2800, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'OIL_GAS', 'MRDM', 'Meridian Midstream',             'FIXED', 2000000,  900000, 4500, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'OIL_GAS', 'FRGT', 'Frostgate LNG',                  'FIXED', 1800000,  800000, 5000, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'OIL_GAS', 'IRWL', 'Ironwell Oilfield Services',     'FIXED', 2400000, 1100000, 2000, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'OIL_GAS', 'KLDP', 'Kaldreth National Petroleum',    'FIXED', 4000000, 2000000, 6000, 'ACTIVE'),
    -- Semiconductors (5)
    ('SYSTEM_COMPANY', 'SEMICONDUCTOR', 'LMFB', 'Lumenfab Foundries',       'FIXED', 3200000, 1600000, 5500, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'SEMICONDUCTOR', 'CRCT', 'Circuiton Design',         'FIXED', 2000000,  900000, 4200, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'SEMICONDUCTOR', 'WFRW', 'Waferworks Equipment',     'FIXED', 1600000,  700000, 4800, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'SEMICONDUCTOR', 'BYTF', 'Byteforge Memory',         'FIXED', 2400000, 1100000, 3300, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'SEMICONDUCTOR', 'NXPL', 'Nexplay Electronics',      'FIXED', 3000000, 1400000, 3800, 'ACTIVE'),
    -- Metals & Mining (4)
    ('SYSTEM_COMPANY', 'METALS', 'AUXA', 'Aurexa Precious Metals',         'FIXED', 1800000,  900000, 6500, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'METALS', 'FERL', 'Ferroline Industrial Metals',    'FIXED', 2600000, 1200000, 2400, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'METALS', 'VLTE', 'Voltearth Materials',            'FIXED', 1800000,  800000, 5200, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'METALS', 'BDRK', 'Bedrock Mining Equipment',       'FIXED', 1400000,  600000, 3000, 'ACTIVE'),
    -- Utilities/Energy (4)
    ('SYSTEM_COMPANY', 'UTILITIES', 'KLWT', 'Kilowatt Power Generation',   'FIXED', 3000000, 1500000, 2000, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'UTILITIES', 'SNFD', 'Sunfield Renewables',         'FIXED', 2200000, 1000000, 2700, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'UTILITIES', 'CLFW', 'Clearflow Water Utilities',   'FIXED', 2000000,  900000, 1600, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'UTILITIES', 'GRDL', 'Gridline Transmission',       'FIXED', 2400000, 1100000, 2200, 'ACTIVE'),
    -- Shipping/Logistics (3)
    ('SYSTEM_COMPANY', 'SHIPPING', 'FRHV', 'Farhaven Container Lines',     'FIXED', 2600000, 1200000, 4600, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'SHIPPING', 'SWLN', 'Swiftlane Logistics',          'FIXED', 2200000, 1000000, 1900, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'SHIPPING', 'PTMK', 'Portmark Harbors',             'FIXED', 1800000,  800000, 2500, 'ACTIVE'),
    -- Misc commodities (2)
    ('SYSTEM_COMPANY', 'MISC_COMMODITIES', 'CTNF', 'Cottonfield Textiles', 'FIXED', 2000000,  900000, 1200, 'ACTIVE'),
    ('SYSTEM_COMPANY', 'MISC_COMMODITIES', 'EVGT', 'Evergreen Timber',     'FIXED', 1600000,  700000, 1700, 'ACTIVE');

-- NAV5: derived-price INDEX card, mintable/burnable like an UNLIMITED card (§5.2/§4.4).
-- Top 5 by market cap (current_price * circulating_supply) among the 30 above:
--   KLDP 12,000,000,000  LMFB 8,800,000,000  AUXA 5,850,000,000  FRHV 5,520,000,000  NXPL 5,320,000,000
-- current_price = sum(component.current_price * weight), weight = component cap / sum(top-5 caps).
INSERT INTO cards (card_type, sector, symbol, name, supply_model, total_supply, circulating_supply, current_price, status) VALUES
    ('INDEX', NULL, 'NAV5', 'NavXchange 5 Index', 'UNLIMITED', NULL, 0, 5442, 'ACTIVE');

INSERT INTO index_components (index_card_id, component_card_id, weight)
SELECT n.id, c.id, v.weight
FROM (VALUES
    ('KLDP', 0.320085),
    ('LMFB', 0.234729),
    ('AUXA', 0.156042),
    ('FRHV', 0.147239),
    ('NXPL', 0.141905)
) AS v(symbol, weight)
JOIN cards c ON c.symbol = v.symbol
CROSS JOIN (SELECT id FROM cards WHERE symbol = 'NAV5') n;
