ALTER TABLE "transfers"
ADD CONSTRAINT transfers_amount_positive CHECK (amount > 0);