INSERT INTO currencies (code, name, symbol) VALUES
-- Major Global Currencies
('USD', 'United States Dollar', '$'),
('EUR', 'Euro', '€'),
('GBP', 'British Pound Sterling', '£'),
('JPY', 'Japanese Yen', '¥'),
('CHF', 'Swiss Franc', 'CHF'),
('CAD', 'Canadian Dollar', 'C$'),
('AUD', 'Australian Dollar', 'A$'),
('NZD', 'New Zealand Dollar', 'NZ$'),

-- Asian Currencies
('SGD', 'Singapore Dollar', 'S$'),
('MYR', 'Malaysian Ringgit', 'RM'),
('IDR', 'Indonesian Rupiah', 'Rp'),
('THB', 'Thai Baht', '฿'),
('VND', 'Vietnamese Dong', '₫'),
('PHP', 'Philippine Peso', '₱'),
('KRW', 'South Korean Won', '₩'),
('CNY', 'Chinese Yuan', '元'),
('HKD', 'Hong Kong Dollar', 'HK$'),
('TWD', 'New Taiwan Dollar', 'NT$'),
('INR', 'Indian Rupee', '₹'),
('PKR', 'Pakistani Rupee', '₨'),
('BDT', 'Bangladeshi Taka', '৳'),
('LKR', 'Sri Lankan Rupee', 'Rs'),

-- Middle East & Africa
('AED', 'UAE Dirham', 'د.إ'),
('SAR', 'Saudi Riyal', '﷼'),
('ILS', 'Israeli New Shekel', '₪'),
('ZAR', 'South African Rand', 'R'),
('EGP', 'Egyptian Pound', 'E£'),
('TRY', 'Turkish Lira', '₺'),

-- Americas
('BRL', 'Brazilian Real', 'R$'),
('MXN', 'Mexican Peso', '$'),
('ARS', 'Argentine Peso', '$'),
('CLP', 'Chilean Peso', '$'),
('COP', 'Colombian Peso', '$'),

-- Others
('RUB', 'Russian Ruble', '₽'),
('SEK', 'Swedish Krona', 'kr'),
('NOK', 'Norwegian Krone', 'kr'),
('DKK', 'Danish Krone', 'kr'),
('PLN', 'Polish Zloty', 'zł')
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    symbol = EXCLUDED.symbol;