-- +goose Up
ALTER TABLE focused_page_deliveries
    DROP CONSTRAINT focused_page_deliveries_tenant_id_focused_page_public_id_t_fkey;

ALTER TABLE focused_page_deliveries
    ADD CONSTRAINT focused_page_deliveries_focused_page_fkey
    FOREIGN KEY (tenant_id, focused_page_public_id, turn_id)
    REFERENCES focused_pages(tenant_id, public_id, turn_id)
    ON DELETE CASCADE;

-- +goose Down
ALTER TABLE focused_page_deliveries
    DROP CONSTRAINT focused_page_deliveries_focused_page_fkey;

ALTER TABLE focused_page_deliveries
    ADD CONSTRAINT focused_page_deliveries_tenant_id_focused_page_public_id_t_fkey
    FOREIGN KEY (tenant_id, focused_page_public_id, turn_id)
    REFERENCES focused_pages(tenant_id, public_id, turn_id);
