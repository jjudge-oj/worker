UPDATE testcases SET input = '' WHERE input IS NULL;
UPDATE testcases SET output = '' WHERE output IS NULL;

ALTER TABLE testcases
    ALTER COLUMN input SET NOT NULL,
    ALTER COLUMN output SET NOT NULL;
