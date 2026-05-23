--
-- PostgreSQL database dump
--


-- Dumped from database version 18.1
-- Dumped by pg_dump version 18.1




--
-- Name: accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.accounts (
    id integer NOT NULL,
    name character varying(255) NOT NULL,
    type character varying(50) NOT NULL,
    category character varying(100) NOT NULL,
    owner_user_id integer,
    currency character varying(10) NOT NULL,
    account_wrapper character varying(50),
    purpose character varying(50) NOT NULL,
    square_meters numeric(10,2),
    is_active boolean NOT NULL,
    receives_contributions boolean NOT NULL,
    created_at timestamp without time zone NOT NULL
);


--
-- Name: accounts_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.accounts_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: accounts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.accounts_id_seq OWNED BY public.accounts.id;


--
-- Name: alembic_version; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.alembic_version (
    version_num character varying(32) NOT NULL
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--
-- Kept in sync with auth.Store.EnsureSchema, which runs the same CREATE TABLE
-- IF NOT EXISTS against pre-existing databases. Declared here so the
-- owner_user_id foreign keys below resolve when schema.sql bootstraps an
-- empty database.

CREATE TABLE public.users (
    id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username character varying(100) NOT NULL UNIQUE,
    password_hash text NOT NULL,
    is_admin boolean NOT NULL DEFAULT false,
    name character varying(100),
    surname character varying(100),
    ppk_employee_rate numeric(5,2),
    ppk_employer_rate numeric(5,2),
    created_at timestamp without time zone NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
);


--
-- Name: app_config; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.app_config (
    id integer NOT NULL,
    birth_date date NOT NULL,
    retirement_age integer NOT NULL,
    retirement_monthly_salary numeric(15,2) NOT NULL,
    allocation_real_estate integer NOT NULL,
    allocation_stocks integer NOT NULL,
    allocation_bonds integer NOT NULL,
    allocation_gold integer NOT NULL,
    allocation_commodities integer NOT NULL,
    monthly_expenses numeric(15,2) NOT NULL,
    monthly_mortgage_payment numeric(15,2) NOT NULL,
    CONSTRAINT single_row CHECK ((id = 1))
);


--
-- Name: app_config_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.app_config_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: app_config_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.app_config_id_seq OWNED BY public.app_config.id;


--
-- Name: assets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.assets (
    id integer NOT NULL,
    name character varying(255) NOT NULL,
    is_active boolean NOT NULL,
    created_at timestamp without time zone NOT NULL
);


--
-- Name: assets_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.assets_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: assets_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.assets_id_seq OWNED BY public.assets.id;


--
-- Name: bonus_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.bonus_events (
    id integer NOT NULL,
    date date NOT NULL,
    amount numeric(15,2) NOT NULL,
    currency character varying(3) NOT NULL,
    type character varying(20) NOT NULL,
    company character varying(200) NOT NULL,
    owner_user_id integer,
    contract_type character varying(50) NOT NULL,
    notes character varying(500),
    is_active boolean NOT NULL,
    created_at timestamp without time zone NOT NULL
);


--
-- Name: bonus_events_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.bonus_events_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: bonus_events_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.bonus_events_id_seq OWNED BY public.bonus_events.id;


--
-- Name: company_valuations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.company_valuations (
    id integer NOT NULL,
    company character varying(200) NOT NULL,
    date date NOT NULL,
    currency character varying(3) NOT NULL,
    fmv_per_share numeric(15,4) NOT NULL,
    fmv_low numeric(15,4),
    fmv_high numeric(15,4),
    source character varying(30) NOT NULL,
    common_stock_discount_pct numeric(5,2),
    notes character varying(500),
    is_active boolean NOT NULL,
    created_at timestamp without time zone NOT NULL
);


--
-- Name: company_valuations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.company_valuations_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: company_valuations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.company_valuations_id_seq OWNED BY public.company_valuations.id;


--
-- Name: cpi_index; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cpi_index (
    year integer NOT NULL,
    yoy_rate numeric(8,4) NOT NULL,
    source character varying(64) NOT NULL,
    fetched_at timestamp with time zone NOT NULL
);


--
-- Name: cpi_index_year_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.cpi_index_year_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: cpi_index_year_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.cpi_index_year_seq OWNED BY public.cpi_index.year;


--
-- Name: debt_payments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.debt_payments (
    id integer NOT NULL,
    account_id integer NOT NULL,
    amount numeric(15,2) NOT NULL,
    date date NOT NULL,
    owner_user_id integer,
    is_active boolean NOT NULL,
    created_at timestamp without time zone NOT NULL
);


--
-- Name: debt_payments_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.debt_payments_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: debt_payments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.debt_payments_id_seq OWNED BY public.debt_payments.id;


--
-- Name: debts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.debts (
    id integer NOT NULL,
    account_id integer NOT NULL,
    name character varying(255) NOT NULL,
    debt_type character varying(50) NOT NULL,
    start_date date NOT NULL,
    initial_amount numeric(15,2) NOT NULL,
    interest_rate numeric(5,2) NOT NULL,
    currency character varying(10) NOT NULL,
    notes character varying(500),
    is_active boolean NOT NULL,
    created_at timestamp without time zone NOT NULL
);


--
-- Name: debts_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.debts_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: debts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.debts_id_seq OWNED BY public.debts.id;


--
-- Name: equity_grants; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.equity_grants (
    id integer NOT NULL,
    grant_date date NOT NULL,
    type character varying(20) NOT NULL,
    company character varying(200) NOT NULL,
    owner_user_id integer,
    total_shares integer NOT NULL,
    strike_price numeric(15,4),
    currency character varying(3) NOT NULL,
    vest_start_date date NOT NULL,
    vest_cliff_months integer NOT NULL,
    vest_total_months integer NOT NULL,
    vest_frequency character varying(20) NOT NULL,
    vest_custom_schedule json,
    requires_liquidity_event boolean NOT NULL,
    liquidity_event_date date,
    tax_treatment character varying(30) NOT NULL,
    notes character varying(500),
    is_active boolean NOT NULL,
    created_at timestamp without time zone NOT NULL
);


--
-- Name: equity_grants_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.equity_grants_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: equity_grants_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.equity_grants_id_seq OWNED BY public.equity_grants.id;


--
-- Name: fx_rates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.fx_rates (
    id integer NOT NULL,
    date date NOT NULL,
    currency character varying(3) NOT NULL,
    rate_pln numeric(15,6) NOT NULL,
    created_at timestamp without time zone NOT NULL
);


--
-- Name: fx_rates_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.fx_rates_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: fx_rates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.fx_rates_id_seq OWNED BY public.fx_rates.id;


--
-- Name: goals; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.goals (
    id integer NOT NULL,
    name character varying(255) NOT NULL,
    target_amount numeric(15,2) NOT NULL,
    target_date date NOT NULL,
    current_amount numeric(15,2) NOT NULL,
    monthly_contribution numeric(15,2) NOT NULL,
    is_completed boolean NOT NULL,
    account_id integer,
    category character varying(100),
    created_at timestamp without time zone NOT NULL
);


--
-- Name: goals_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.goals_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: goals_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.goals_id_seq OWNED BY public.goals.id;


--
-- Name: retirement_limits; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.retirement_limits (
    id integer NOT NULL,
    year integer NOT NULL,
    account_wrapper character varying(10) NOT NULL,
    owner_user_id integer,
    limit_amount numeric(15,2) NOT NULL,
    notes character varying(255)
);


--
-- Name: retirement_limits_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.retirement_limits_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: retirement_limits_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.retirement_limits_id_seq OWNED BY public.retirement_limits.id;


--
-- Name: salary_records; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.salary_records (
    id integer NOT NULL,
    date date NOT NULL,
    gross_amount numeric(15,2) NOT NULL,
    contract_type character varying(50) NOT NULL,
    company character varying(200) NOT NULL,
    owner_user_id integer,
    is_active boolean NOT NULL,
    created_at timestamp without time zone NOT NULL
);


--
-- Name: salary_records_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.salary_records_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: salary_records_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.salary_records_id_seq OWNED BY public.salary_records.id;


--
-- Name: snapshot_aggregates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.snapshot_aggregates (
    id integer NOT NULL,
    snapshot_id integer NOT NULL,
    month date NOT NULL,
    owner_user_id integer,
    total_assets numeric(15,2) NOT NULL,
    total_liabilities numeric(15,2) NOT NULL,
    net_worth numeric(15,2) NOT NULL,
    allocation_json json NOT NULL,
    computed_at timestamp with time zone NOT NULL
);


--
-- Name: snapshot_aggregates_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.snapshot_aggregates_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: snapshot_aggregates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.snapshot_aggregates_id_seq OWNED BY public.snapshot_aggregates.id;


--
-- Name: snapshot_values; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.snapshot_values (
    id integer NOT NULL,
    snapshot_id integer NOT NULL,
    asset_id integer,
    account_id integer,
    value numeric(15,2) NOT NULL,
    CONSTRAINT ck_asset_or_account CHECK ((((asset_id IS NOT NULL) AND (account_id IS NULL)) OR ((asset_id IS NULL) AND (account_id IS NOT NULL))))
);


--
-- Name: snapshot_values_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.snapshot_values_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: snapshot_values_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.snapshot_values_id_seq OWNED BY public.snapshot_values.id;


--
-- Name: snapshots; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.snapshots (
    id integer NOT NULL,
    date date NOT NULL,
    notes text,
    created_at timestamp without time zone NOT NULL
);


--
-- Name: snapshots_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.snapshots_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: snapshots_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.snapshots_id_seq OWNED BY public.snapshots.id;


--
-- Name: transactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.transactions (
    id integer NOT NULL,
    account_id integer NOT NULL,
    amount numeric(15,2) NOT NULL,
    date date NOT NULL,
    owner_user_id integer,
    transaction_type character varying(20),
    is_active boolean NOT NULL,
    created_at timestamp without time zone NOT NULL
);


--
-- Name: transactions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.transactions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: transactions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.transactions_id_seq OWNED BY public.transactions.id;


--
-- Name: accounts id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.accounts ALTER COLUMN id SET DEFAULT nextval('public.accounts_id_seq'::regclass);


--
-- Name: app_config id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_config ALTER COLUMN id SET DEFAULT nextval('public.app_config_id_seq'::regclass);


--
-- Name: assets id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assets ALTER COLUMN id SET DEFAULT nextval('public.assets_id_seq'::regclass);


--
-- Name: bonus_events id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bonus_events ALTER COLUMN id SET DEFAULT nextval('public.bonus_events_id_seq'::regclass);


--
-- Name: company_valuations id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.company_valuations ALTER COLUMN id SET DEFAULT nextval('public.company_valuations_id_seq'::regclass);


--
-- Name: cpi_index year; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cpi_index ALTER COLUMN year SET DEFAULT nextval('public.cpi_index_year_seq'::regclass);


--
-- Name: debt_payments id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.debt_payments ALTER COLUMN id SET DEFAULT nextval('public.debt_payments_id_seq'::regclass);


--
-- Name: debts id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.debts ALTER COLUMN id SET DEFAULT nextval('public.debts_id_seq'::regclass);


--
-- Name: equity_grants id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.equity_grants ALTER COLUMN id SET DEFAULT nextval('public.equity_grants_id_seq'::regclass);


--
-- Name: fx_rates id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.fx_rates ALTER COLUMN id SET DEFAULT nextval('public.fx_rates_id_seq'::regclass);


--
-- Name: goals id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.goals ALTER COLUMN id SET DEFAULT nextval('public.goals_id_seq'::regclass);


--
-- Name: retirement_limits id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.retirement_limits ALTER COLUMN id SET DEFAULT nextval('public.retirement_limits_id_seq'::regclass);


--
-- Name: salary_records id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.salary_records ALTER COLUMN id SET DEFAULT nextval('public.salary_records_id_seq'::regclass);


--
-- Name: snapshot_aggregates id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.snapshot_aggregates ALTER COLUMN id SET DEFAULT nextval('public.snapshot_aggregates_id_seq'::regclass);


--
-- Name: snapshot_values id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.snapshot_values ALTER COLUMN id SET DEFAULT nextval('public.snapshot_values_id_seq'::regclass);


--
-- Name: snapshots id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.snapshots ALTER COLUMN id SET DEFAULT nextval('public.snapshots_id_seq'::regclass);


--
-- Name: transactions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transactions ALTER COLUMN id SET DEFAULT nextval('public.transactions_id_seq'::regclass);


--
-- Name: accounts accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_pkey PRIMARY KEY (id);


--
-- Name: alembic_version alembic_version_pkc; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alembic_version
    ADD CONSTRAINT alembic_version_pkc PRIMARY KEY (version_num);


--
-- Name: app_config app_config_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_config
    ADD CONSTRAINT app_config_pkey PRIMARY KEY (id);


--
-- Name: assets assets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT assets_pkey PRIMARY KEY (id);


--
-- Name: bonus_events bonus_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bonus_events
    ADD CONSTRAINT bonus_events_pkey PRIMARY KEY (id);


--
-- Name: company_valuations company_valuations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.company_valuations
    ADD CONSTRAINT company_valuations_pkey PRIMARY KEY (id);


--
-- Name: cpi_index cpi_index_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cpi_index
    ADD CONSTRAINT cpi_index_pkey PRIMARY KEY (year);


--
-- Name: debt_payments debt_payments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.debt_payments
    ADD CONSTRAINT debt_payments_pkey PRIMARY KEY (id);


--
-- Name: debts debts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.debts
    ADD CONSTRAINT debts_pkey PRIMARY KEY (id);


--
-- Name: equity_grants equity_grants_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.equity_grants
    ADD CONSTRAINT equity_grants_pkey PRIMARY KEY (id);


--
-- Name: fx_rates fx_rates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.fx_rates
    ADD CONSTRAINT fx_rates_pkey PRIMARY KEY (id);


--
-- Name: goals goals_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.goals
    ADD CONSTRAINT goals_pkey PRIMARY KEY (id);


--
-- Name: retirement_limits retirement_limits_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.retirement_limits
    ADD CONSTRAINT retirement_limits_pkey PRIMARY KEY (id);


--
-- Name: salary_records salary_records_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.salary_records
    ADD CONSTRAINT salary_records_pkey PRIMARY KEY (id);


--
-- Name: snapshot_aggregates snapshot_aggregates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.snapshot_aggregates
    ADD CONSTRAINT snapshot_aggregates_pkey PRIMARY KEY (id);


--
-- Name: snapshot_values snapshot_values_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.snapshot_values
    ADD CONSTRAINT snapshot_values_pkey PRIMARY KEY (id);


--
-- Name: snapshots snapshots_date_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.snapshots
    ADD CONSTRAINT snapshots_date_key UNIQUE (date);


--
-- Name: snapshots snapshots_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.snapshots
    ADD CONSTRAINT snapshots_pkey PRIMARY KEY (id);


--
-- Name: transactions transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_pkey PRIMARY KEY (id);


--
-- Name: snapshot_values uix_snapshot_account; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.snapshot_values
    ADD CONSTRAINT uix_snapshot_account UNIQUE (snapshot_id, account_id);


--
-- Name: snapshot_aggregates uix_snapshot_agg_snapshot_owner; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.snapshot_aggregates
    ADD CONSTRAINT uix_snapshot_agg_snapshot_owner UNIQUE NULLS NOT DISTINCT (snapshot_id, owner_user_id);


--
-- Name: snapshot_values uix_snapshot_asset; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.snapshot_values
    ADD CONSTRAINT uix_snapshot_asset UNIQUE (snapshot_id, asset_id);


--
-- Name: debts uq_debt_account; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.debts
    ADD CONSTRAINT uq_debt_account UNIQUE (account_id);


--
-- Name: fx_rates uq_fx_rates_date_currency; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.fx_rates
    ADD CONSTRAINT uq_fx_rates_date_currency UNIQUE (date, currency);


--
-- Name: retirement_limits uq_year_wrapper_owner; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.retirement_limits
    ADD CONSTRAINT uq_year_wrapper_owner UNIQUE NULLS NOT DISTINCT (year, account_wrapper, owner_user_id);


--
-- Name: ix_snapshot_aggregates_month; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_snapshot_aggregates_month ON public.snapshot_aggregates USING btree (month);


--
-- Name: ix_snapshot_values_asset_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_snapshot_values_asset_id ON public.snapshot_values USING btree (asset_id);


--
-- Name: ix_transactions_account_id_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_transactions_account_id_date ON public.transactions USING btree (account_id, date);


--
-- Name: debt_payments debt_payments_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.debt_payments
    ADD CONSTRAINT debt_payments_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id) ON DELETE CASCADE;


--
-- Name: debts debts_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.debts
    ADD CONSTRAINT debts_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id) ON DELETE CASCADE;


--
-- Name: goals goals_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.goals
    ADD CONSTRAINT goals_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id);


--
-- Name: snapshot_aggregates snapshot_aggregates_snapshot_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.snapshot_aggregates
    ADD CONSTRAINT snapshot_aggregates_snapshot_id_fkey FOREIGN KEY (snapshot_id) REFERENCES public.snapshots(id) ON DELETE CASCADE;


--
-- Name: snapshot_values snapshot_values_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.snapshot_values
    ADD CONSTRAINT snapshot_values_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id) ON DELETE CASCADE;


--
-- Name: snapshot_values snapshot_values_asset_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.snapshot_values
    ADD CONSTRAINT snapshot_values_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES public.assets(id) ON DELETE CASCADE;


--
-- Name: snapshot_values snapshot_values_snapshot_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.snapshot_values
    ADD CONSTRAINT snapshot_values_snapshot_id_fkey FOREIGN KEY (snapshot_id) REFERENCES public.snapshots(id) ON DELETE CASCADE;


--
-- Name: transactions transactions_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id) ON DELETE CASCADE;


--
-- Name: owner_user_id foreign keys; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_owner_user_id_fkey FOREIGN KEY (owner_user_id) REFERENCES public.users(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_owner_user_id_fkey FOREIGN KEY (owner_user_id) REFERENCES public.users(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.salary_records
    ADD CONSTRAINT salary_records_owner_user_id_fkey FOREIGN KEY (owner_user_id) REFERENCES public.users(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.debt_payments
    ADD CONSTRAINT debt_payments_owner_user_id_fkey FOREIGN KEY (owner_user_id) REFERENCES public.users(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.bonus_events
    ADD CONSTRAINT bonus_events_owner_user_id_fkey FOREIGN KEY (owner_user_id) REFERENCES public.users(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.equity_grants
    ADD CONSTRAINT equity_grants_owner_user_id_fkey FOREIGN KEY (owner_user_id) REFERENCES public.users(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.retirement_limits
    ADD CONSTRAINT retirement_limits_owner_user_id_fkey FOREIGN KEY (owner_user_id) REFERENCES public.users(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.snapshot_aggregates
    ADD CONSTRAINT snapshot_aggregates_owner_user_id_fkey FOREIGN KEY (owner_user_id) REFERENCES public.users(id) ON DELETE RESTRICT;


--
-- Name: treasury_bonds; Type: TABLE; Schema: public; Owner: -
--
-- Kept in sync with bonds.Store.EnsureSchema, which runs the same CREATE
-- TABLE IF NOT EXISTS against pre-existing databases.

CREATE TABLE public.treasury_bonds (
    id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    type character varying(8) NOT NULL,
    series character varying(64) NOT NULL,
    face_value numeric(15,2) NOT NULL,
    purchase_date date NOT NULL,
    owner_user_id integer,
    first_year_rate numeric(8,4) NOT NULL,
    margin numeric(8,4) NOT NULL DEFAULT 0,
    capitalize boolean NOT NULL DEFAULT true,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp without time zone NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
);


ALTER TABLE ONLY public.treasury_bonds
    ADD CONSTRAINT treasury_bonds_owner_user_id_fkey FOREIGN KEY (owner_user_id) REFERENCES public.users(id) ON DELETE RESTRICT;


--
-- PostgreSQL database dump complete
--


