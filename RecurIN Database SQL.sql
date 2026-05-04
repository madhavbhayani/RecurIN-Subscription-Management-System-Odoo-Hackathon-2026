--
-- PostgreSQL database dump
--

\restrict rJxpRw7lM4JiLxrJep5zUdGAeHw9Sxxi6FWUWdg2fZeR4ixyPcEM1qg3eXZxePm

-- Dumped from database version 17.6
-- Dumped by pg_dump version 17.6

-- Started on 2026-05-04 18:23:00

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- TOC entry 8 (class 2615 OID 26842)
-- Name: attributes; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA attributes;


ALTER SCHEMA attributes OWNER TO postgres;

--
-- TOC entry 14 (class 2615 OID 27091)
-- Name: discount; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA discount;


ALTER SCHEMA discount OWNER TO postgres;

--
-- TOC entry 13 (class 2615 OID 27075)
-- Name: payment_term; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA payment_term;


ALTER SCHEMA payment_term OWNER TO postgres;

--
-- TOC entry 16 (class 2615 OID 27297)
-- Name: privileges; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA privileges;


ALTER SCHEMA privileges OWNER TO postgres;

--
-- TOC entry 10 (class 2615 OID 26889)
-- Name: products; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA products;


ALTER SCHEMA products OWNER TO postgres;

--
-- TOC entry 12 (class 2615 OID 27007)
-- Name: quotations; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA quotations;


ALTER SCHEMA quotations OWNER TO postgres;

--
-- TOC entry 11 (class 2615 OID 26964)
-- Name: recurring_plans; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA recurring_plans;


ALTER SCHEMA recurring_plans OWNER TO postgres;

--
-- TOC entry 15 (class 2615 OID 27149)
-- Name: subscription; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA subscription;


ALTER SCHEMA subscription OWNER TO postgres;

--
-- TOC entry 9 (class 2615 OID 26872)
-- Name: taxes; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA taxes;


ALTER SCHEMA taxes OWNER TO postgres;

--
-- TOC entry 7 (class 2615 OID 26817)
-- Name: users; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA users;


ALTER SCHEMA users OWNER TO postgres;

--
-- TOC entry 2 (class 3079 OID 26780)
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- TOC entry 5220 (class 0 OID 0)
-- Dependencies: 2
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


--
-- TOC entry 989 (class 1247 OID 27853)
-- Name: user_role; Type: TYPE; Schema: users; Owner: postgres
--

CREATE TYPE users.user_role AS ENUM (
    'Admin',
    'Internal',
    'User',
    'Manager',
    'Support'
);


ALTER TYPE users.user_role OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- TOC entry 230 (class 1259 OID 26843)
-- Name: attribute; Type: TABLE; Schema: attributes; Owner: postgres
--

CREATE TABLE attributes.attribute (
    attribute_id uuid DEFAULT gen_random_uuid() NOT NULL,
    attribute_name character varying(150) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE attributes.attribute OWNER TO postgres;

--
-- TOC entry 231 (class 1259 OID 26853)
-- Name: attribute_values; Type: TABLE; Schema: attributes; Owner: postgres
--

CREATE TABLE attributes.attribute_values (
    attribute_value_id uuid DEFAULT gen_random_uuid() NOT NULL,
    attribute_id uuid NOT NULL,
    attribute_value character varying(200) NOT NULL,
    default_extra_price numeric(12,2) DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_attribute_values_default_extra_price_non_negative CHECK ((default_extra_price >= (0)::numeric))
);


ALTER TABLE attributes.attribute_values OWNER TO postgres;

--
-- TOC entry 235 (class 1259 OID 26947)
-- Name: product_variants; Type: TABLE; Schema: attributes; Owner: postgres
--

CREATE TABLE attributes.product_variants (
    product_id uuid NOT NULL,
    attribute_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE attributes.product_variants OWNER TO postgres;

--
-- TOC entry 239 (class 1259 OID 27092)
-- Name: discount_data; Type: TABLE; Schema: discount; Owner: postgres
--

CREATE TABLE discount.discount_data (
    discount_id uuid DEFAULT gen_random_uuid() NOT NULL,
    discount_name character varying(180) NOT NULL,
    discount_unit character varying(20) NOT NULL,
    discount_value numeric(12,2) NOT NULL,
    minimum_purchase numeric(12,2) DEFAULT 0 NOT NULL,
    maximum_purchase numeric(12,2) NOT NULL,
    start_date date NOT NULL,
    end_date date NOT NULL,
    is_limit boolean DEFAULT false NOT NULL,
    limit_users integer,
    applied_user_count integer DEFAULT 0 NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_discount_applied_user_count CHECK ((applied_user_count >= 0)),
    CONSTRAINT chk_discount_limit_rule CHECK ((((is_limit = true) AND (limit_users IS NOT NULL) AND (limit_users > 0)) OR ((is_limit = false) AND (limit_users IS NULL)))),
    CONSTRAINT chk_discount_percentage_value CHECK ((((discount_unit)::text <> 'Percentage'::text) OR (discount_value <= (100)::numeric))),
    CONSTRAINT chk_discount_purchase_values CHECK (((minimum_purchase >= (0)::numeric) AND (maximum_purchase >= minimum_purchase))),
    CONSTRAINT chk_discount_value_positive CHECK ((discount_value > (0)::numeric))
);


ALTER TABLE discount.discount_data OWNER TO postgres;

--
-- TOC entry 240 (class 1259 OID 27113)
-- Name: discount_products; Type: TABLE; Schema: discount; Owner: postgres
--

CREATE TABLE discount.discount_products (
    discount_id uuid NOT NULL,
    product_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE discount.discount_products OWNER TO postgres;

--
-- TOC entry 245 (class 1259 OID 27219)
-- Name: product_discount; Type: TABLE; Schema: discount; Owner: postgres
--

CREATE TABLE discount.product_discount (
    product_id uuid NOT NULL,
    discount_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE discount.product_discount OWNER TO postgres;

--
-- TOC entry 238 (class 1259 OID 27076)
-- Name: payment_term_data; Type: TABLE; Schema: payment_term; Owner: postgres
--

CREATE TABLE payment_term.payment_term_data (
    payment_term_id uuid DEFAULT gen_random_uuid() NOT NULL,
    payment_term_name character varying(180) NOT NULL,
    due_unit character varying(20) NOT NULL,
    due_value numeric(12,2) NOT NULL,
    interval_days integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_payment_term_due_value_percentage_limit CHECK ((((due_unit)::text <> 'Percentage'::text) OR (due_value <= (100)::numeric))),
    CONSTRAINT chk_payment_term_due_value_positive CHECK ((due_value > (0)::numeric)),
    CONSTRAINT chk_payment_term_interval_days_positive CHECK ((interval_days > 0))
);


ALTER TABLE payment_term.payment_term_data OWNER TO postgres;

--
-- TOC entry 248 (class 1259 OID 27298)
-- Name: role_data; Type: TABLE; Schema: privileges; Owner: postgres
--

CREATE TABLE privileges.role_data (
    role_id uuid DEFAULT gen_random_uuid() NOT NULL,
    role_name character varying(150) NOT NULL,
    user_id uuid,
    is_system boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_privileges_role_name_not_empty CHECK ((btrim((role_name)::text) <> ''::text))
);


ALTER TABLE privileges.role_data OWNER TO postgres;

--
-- TOC entry 249 (class 1259 OID 27318)
-- Name: role_permissions; Type: TABLE; Schema: privileges; Owner: postgres
--

CREATE TABLE privileges.role_permissions (
    role_permission_id uuid DEFAULT gen_random_uuid() NOT NULL,
    role_id uuid NOT NULL,
    resource_key character varying(120) NOT NULL,
    can_create boolean DEFAULT false NOT NULL,
    can_read boolean DEFAULT false NOT NULL,
    can_update boolean DEFAULT false NOT NULL,
    can_delete boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_privileges_permissions_resource_key_not_empty CHECK ((btrim((resource_key)::text) <> ''::text))
);


ALTER TABLE privileges.role_permissions OWNER TO postgres;

--
-- TOC entry 233 (class 1259 OID 26890)
-- Name: product_data; Type: TABLE; Schema: products; Owner: postgres
--

CREATE TABLE products.product_data (
    product_id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_name character varying(180) NOT NULL,
    product_type character varying(20) NOT NULL,
    sales_price numeric(12,2) DEFAULT 0 NOT NULL,
    cost_price numeric(12,2) DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    recurring_plan_id uuid,
    CONSTRAINT chk_product_cost_price_non_negative CHECK ((cost_price >= (0)::numeric)),
    CONSTRAINT chk_product_sales_price_non_negative CHECK ((sales_price >= (0)::numeric))
);


ALTER TABLE products.product_data OWNER TO postgres;

--
-- TOC entry 228 (class 1259 OID 26772)
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    name text NOT NULL,
    applied_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.schema_migrations OWNER TO postgres;

--
-- TOC entry 237 (class 1259 OID 27008)
-- Name: quotation; Type: TABLE; Schema: quotations; Owner: postgres
--

CREATE TABLE quotations.quotation (
    quotation_id uuid DEFAULT gen_random_uuid() NOT NULL,
    last_forever boolean DEFAULT false NOT NULL,
    quotation_validity_days integer,
    recurring_plan_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_quotation_validity_days CHECK ((((last_forever = true) AND (quotation_validity_days IS NULL)) OR ((last_forever = false) AND (quotation_validity_days IS NOT NULL) AND (quotation_validity_days > 0))))
);


ALTER TABLE quotations.quotation OWNER TO postgres;

--
-- TOC entry 241 (class 1259 OID 27132)
-- Name: quotations_products; Type: TABLE; Schema: quotations; Owner: postgres
--

CREATE TABLE quotations.quotations_products (
    quotation_id uuid NOT NULL,
    product_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE quotations.quotations_products OWNER TO postgres;

--
-- TOC entry 236 (class 1259 OID 26965)
-- Name: recurring_plan_data; Type: TABLE; Schema: recurring_plans; Owner: postgres
--

CREATE TABLE recurring_plans.recurring_plan_data (
    recurring_plan_id uuid DEFAULT gen_random_uuid() NOT NULL,
    recurring_name character varying(180) NOT NULL,
    billing_period character varying(20) NOT NULL,
    is_closable boolean DEFAULT false NOT NULL,
    automatic_close_cycles integer,
    is_pausable boolean DEFAULT false NOT NULL,
    is_renewable boolean DEFAULT true NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_recurring_plan_automatic_close_cycles_positive CHECK (((automatic_close_cycles IS NULL) OR (automatic_close_cycles > 0))),
    CONSTRAINT chk_recurring_plan_automatic_close_required_when_closable CHECK ((((is_closable = true) AND (automatic_close_cycles IS NOT NULL)) OR ((is_closable = false) AND (automatic_close_cycles IS NULL))))
);


ALTER TABLE recurring_plans.recurring_plan_data OWNER TO postgres;

--
-- TOC entry 243 (class 1259 OID 27173)
-- Name: subscription_number_seq; Type: SEQUENCE; Schema: subscription; Owner: postgres
--

CREATE SEQUENCE subscription.subscription_number_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE subscription.subscription_number_seq OWNER TO postgres;

--
-- TOC entry 246 (class 1259 OID 27242)
-- Name: subscription_other_info; Type: TABLE; Schema: subscription; Owner: postgres
--

CREATE TABLE subscription.subscription_other_info (
    subscription_other_info_id uuid DEFAULT gen_random_uuid() NOT NULL,
    subscription_id uuid NOT NULL,
    sales_person character varying(180),
    start_date date,
    payment_method character varying(120),
    is_payment_mode boolean,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE subscription.subscription_other_info OWNER TO postgres;

--
-- TOC entry 247 (class 1259 OID 27259)
-- Name: subscription_product_variants; Type: TABLE; Schema: subscription; Owner: postgres
--

CREATE TABLE subscription.subscription_product_variants (
    subscription_product_variant_id uuid DEFAULT gen_random_uuid() NOT NULL,
    subscription_product_id uuid NOT NULL,
    product_id uuid NOT NULL,
    attribute_id uuid NOT NULL,
    attribute_value_id uuid NOT NULL,
    extra_price numeric(12,2) DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_subscription_product_variants_extra_price_non_negative CHECK ((extra_price >= (0)::numeric))
);


ALTER TABLE subscription.subscription_product_variants OWNER TO postgres;

--
-- TOC entry 244 (class 1259 OID 27187)
-- Name: subscription_products; Type: TABLE; Schema: subscription; Owner: postgres
--

CREATE TABLE subscription.subscription_products (
    subscription_product_id uuid DEFAULT gen_random_uuid() NOT NULL,
    subscription_id uuid NOT NULL,
    product_id uuid NOT NULL,
    quantity integer DEFAULT 1 NOT NULL,
    unit_price numeric(12,2) DEFAULT 0 NOT NULL,
    discount_amount numeric(12,2) DEFAULT 0 NOT NULL,
    tax_amount numeric(12,2) DEFAULT 0 NOT NULL,
    total_amount numeric(12,2) DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_subscription_products_discount_non_negative CHECK ((discount_amount >= (0)::numeric)),
    CONSTRAINT chk_subscription_products_quantity_positive CHECK ((quantity >= 1)),
    CONSTRAINT chk_subscription_products_tax_non_negative CHECK ((tax_amount >= (0)::numeric)),
    CONSTRAINT chk_subscription_products_total_non_negative CHECK ((total_amount >= (0)::numeric)),
    CONSTRAINT chk_subscription_products_unit_price_non_negative CHECK ((unit_price >= (0)::numeric))
);


ALTER TABLE subscription.subscription_products OWNER TO postgres;

--
-- TOC entry 242 (class 1259 OID 27150)
-- Name: subscriptions; Type: TABLE; Schema: subscription; Owner: postgres
--

CREATE TABLE subscription.subscriptions (
    subscription_id uuid DEFAULT gen_random_uuid() NOT NULL,
    subscription_number character varying(80) DEFAULT ('SUB'::text || lpad((nextval('subscription.subscription_number_seq'::regclass))::text, 6, '0'::text)) NOT NULL,
    customer_name character varying(180) NOT NULL,
    next_invoice_date date NOT NULL,
    recurring_plan_id uuid,
    status character varying(30) DEFAULT 'Quotation Sent'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    customer_id uuid,
    quotation_id uuid,
    payment_term_id uuid,
    CONSTRAINT chk_subscriptions_status CHECK (((status)::text = ANY ((ARRAY['Draft'::character varying, 'Quotation Sent'::character varying, 'Active'::character varying, 'Confirmed'::character varying, 'Cancelled'::character varying])::text[])))
);


ALTER TABLE subscription.subscriptions OWNER TO postgres;

--
-- TOC entry 234 (class 1259 OID 26906)
-- Name: product_tax; Type: TABLE; Schema: taxes; Owner: postgres
--

CREATE TABLE taxes.product_tax (
    product_id uuid NOT NULL,
    tax_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE taxes.product_tax OWNER TO postgres;

--
-- TOC entry 232 (class 1259 OID 26873)
-- Name: tax_data; Type: TABLE; Schema: taxes; Owner: postgres
--

CREATE TABLE taxes.tax_data (
    tax_id uuid DEFAULT gen_random_uuid() NOT NULL,
    tax_name character varying(150) NOT NULL,
    tax_computation_unit character varying(20) NOT NULL,
    tax_computation_value numeric(12,2) DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_tax_computation_value_non_negative CHECK ((tax_computation_value >= (0)::numeric)),
    CONSTRAINT chk_tax_percentage_max CHECK ((((tax_computation_unit)::text <> 'Percentage'::text) OR (tax_computation_value <= (100)::numeric)))
);


ALTER TABLE taxes.tax_data OWNER TO postgres;

--
-- TOC entry 250 (class 1259 OID 27345)
-- Name: cart; Type: TABLE; Schema: users; Owner: postgres
--

CREATE TABLE users.cart (
    cart_item_id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    product_id uuid NOT NULL,
    selected_variant_attribute_id uuid,
    quantity integer DEFAULT 1 NOT NULL,
    unit_price numeric(12,2) DEFAULT 0 NOT NULL,
    selected_variant_price numeric(12,2) DEFAULT 0 NOT NULL,
    discount_amount numeric(12,2) DEFAULT 0 NOT NULL,
    billing_period character varying(20),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_users_cart_discount_non_negative CHECK ((discount_amount >= (0)::numeric)),
    CONSTRAINT chk_users_cart_quantity_positive CHECK ((quantity >= 1)),
    CONSTRAINT chk_users_cart_unit_price_non_negative CHECK ((unit_price >= (0)::numeric)),
    CONSTRAINT chk_users_cart_variant_price_non_negative CHECK ((selected_variant_price >= (0)::numeric))
);


ALTER TABLE users.cart OWNER TO postgres;

--
-- TOC entry 252 (class 1259 OID 27417)
-- Name: password_reset_otp; Type: TABLE; Schema: users; Owner: postgres
--

CREATE TABLE users.password_reset_otp (
    password_reset_otp_id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    email character varying(255) NOT NULL,
    otp_hash text NOT NULL,
    otp_expires_at timestamp with time zone NOT NULL,
    verify_attempts integer DEFAULT 0 NOT NULL,
    verified_at timestamp with time zone,
    reset_token_hash text,
    reset_token_expires_at timestamp with time zone,
    used_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_password_reset_verify_attempts_non_negative CHECK ((verify_attempts >= 0))
);


ALTER TABLE users.password_reset_otp OWNER TO postgres;

--
-- TOC entry 251 (class 1259 OID 27379)
-- Name: payments; Type: TABLE; Schema: users; Owner: postgres
--

CREATE TABLE users.payments (
    payment_id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    subscription_id uuid,
    paypal_payment_id character varying(120) NOT NULL,
    paypal_payer_id character varying(120),
    paypal_capture_id character varying(120),
    paypal_status character varying(50) NOT NULL,
    amount_inr numeric(12,2) DEFAULT 0 NOT NULL,
    amount_usd numeric(12,2) DEFAULT 0 NOT NULL,
    currency_from character varying(10) DEFAULT 'INR'::character varying NOT NULL,
    currency_to character varying(10) DEFAULT 'USD'::character varying NOT NULL,
    payment_method character varying(60) DEFAULT 'PayPal'::character varying NOT NULL,
    payment_date timestamp with time zone DEFAULT now() NOT NULL,
    raw_payload jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_users_payments_amount_inr_non_negative CHECK ((amount_inr >= (0)::numeric)),
    CONSTRAINT chk_users_payments_amount_usd_non_negative CHECK ((amount_usd >= (0)::numeric))
);


ALTER TABLE users.payments OWNER TO postgres;

--
-- TOC entry 229 (class 1259 OID 26825)
-- Name: user; Type: TABLE; Schema: users; Owner: postgres
--

CREATE TABLE users."user" (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(150) NOT NULL,
    phone_number character varying(16) NOT NULL,
    address text,
    email character varying(255) NOT NULL,
    password_hash text NOT NULL,
    role users.user_role DEFAULT 'User'::users.user_role NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_user_phone_e164_format CHECK (((phone_number)::text ~ '^\+[1-9][0-9]{7,14}$'::text))
);


ALTER TABLE users."user" OWNER TO postgres;

--
-- TOC entry 4923 (class 2606 OID 26852)
-- Name: attribute attribute_attribute_name_key; Type: CONSTRAINT; Schema: attributes; Owner: postgres
--

ALTER TABLE ONLY attributes.attribute
    ADD CONSTRAINT attribute_attribute_name_key UNIQUE (attribute_name);


--
-- TOC entry 4925 (class 2606 OID 26850)
-- Name: attribute attribute_pkey; Type: CONSTRAINT; Schema: attributes; Owner: postgres
--

ALTER TABLE ONLY attributes.attribute
    ADD CONSTRAINT attribute_pkey PRIMARY KEY (attribute_id);


--
-- TOC entry 4928 (class 2606 OID 26862)
-- Name: attribute_values attribute_values_pkey; Type: CONSTRAINT; Schema: attributes; Owner: postgres
--

ALTER TABLE ONLY attributes.attribute_values
    ADD CONSTRAINT attribute_values_pkey PRIMARY KEY (attribute_value_id);


--
-- TOC entry 4949 (class 2606 OID 26952)
-- Name: product_variants product_variants_v2_pkey; Type: CONSTRAINT; Schema: attributes; Owner: postgres
--

ALTER TABLE ONLY attributes.product_variants
    ADD CONSTRAINT product_variants_v2_pkey PRIMARY KEY (product_id, attribute_id);


--
-- TOC entry 4931 (class 2606 OID 26924)
-- Name: attribute_values uq_attribute_values_id_attribute; Type: CONSTRAINT; Schema: attributes; Owner: postgres
--

ALTER TABLE ONLY attributes.attribute_values
    ADD CONSTRAINT uq_attribute_values_id_attribute UNIQUE (attribute_value_id, attribute_id);


--
-- TOC entry 4965 (class 2606 OID 27112)
-- Name: discount_data discount_data_discount_name_key; Type: CONSTRAINT; Schema: discount; Owner: postgres
--

ALTER TABLE ONLY discount.discount_data
    ADD CONSTRAINT discount_data_discount_name_key UNIQUE (discount_name);


--
-- TOC entry 4967 (class 2606 OID 27110)
-- Name: discount_data discount_data_pkey; Type: CONSTRAINT; Schema: discount; Owner: postgres
--

ALTER TABLE ONLY discount.discount_data
    ADD CONSTRAINT discount_data_pkey PRIMARY KEY (discount_id);


--
-- TOC entry 4971 (class 2606 OID 27118)
-- Name: discount_products discount_products_pkey; Type: CONSTRAINT; Schema: discount; Owner: postgres
--

ALTER TABLE ONLY discount.discount_products
    ADD CONSTRAINT discount_products_pkey PRIMARY KEY (discount_id, product_id);


--
-- TOC entry 4996 (class 2606 OID 27224)
-- Name: product_discount product_discount_pkey; Type: CONSTRAINT; Schema: discount; Owner: postgres
--

ALTER TABLE ONLY discount.product_discount
    ADD CONSTRAINT product_discount_pkey PRIMARY KEY (product_id, discount_id);


--
-- TOC entry 4961 (class 2606 OID 27089)
-- Name: payment_term_data payment_term_data_payment_term_name_key; Type: CONSTRAINT; Schema: payment_term; Owner: postgres
--

ALTER TABLE ONLY payment_term.payment_term_data
    ADD CONSTRAINT payment_term_data_payment_term_name_key UNIQUE (payment_term_name);


--
-- TOC entry 4963 (class 2606 OID 27087)
-- Name: payment_term_data payment_term_data_pkey; Type: CONSTRAINT; Schema: payment_term; Owner: postgres
--

ALTER TABLE ONLY payment_term.payment_term_data
    ADD CONSTRAINT payment_term_data_pkey PRIMARY KEY (payment_term_id);


--
-- TOC entry 5014 (class 2606 OID 27307)
-- Name: role_data role_data_pkey; Type: CONSTRAINT; Schema: privileges; Owner: postgres
--

ALTER TABLE ONLY privileges.role_data
    ADD CONSTRAINT role_data_pkey PRIMARY KEY (role_id);


--
-- TOC entry 5016 (class 2606 OID 27309)
-- Name: role_data role_data_role_name_key; Type: CONSTRAINT; Schema: privileges; Owner: postgres
--

ALTER TABLE ONLY privileges.role_data
    ADD CONSTRAINT role_data_role_name_key UNIQUE (role_name);


--
-- TOC entry 5018 (class 2606 OID 27311)
-- Name: role_data role_data_user_id_key; Type: CONSTRAINT; Schema: privileges; Owner: postgres
--

ALTER TABLE ONLY privileges.role_data
    ADD CONSTRAINT role_data_user_id_key UNIQUE (user_id);


--
-- TOC entry 5021 (class 2606 OID 27330)
-- Name: role_permissions role_permissions_pkey; Type: CONSTRAINT; Schema: privileges; Owner: postgres
--

ALTER TABLE ONLY privileges.role_permissions
    ADD CONSTRAINT role_permissions_pkey PRIMARY KEY (role_permission_id);


--
-- TOC entry 4941 (class 2606 OID 26902)
-- Name: product_data product_data_pkey; Type: CONSTRAINT; Schema: products; Owner: postgres
--

ALTER TABLE ONLY products.product_data
    ADD CONSTRAINT product_data_pkey PRIMARY KEY (product_id);


--
-- TOC entry 4943 (class 2606 OID 26904)
-- Name: product_data product_data_product_name_key; Type: CONSTRAINT; Schema: products; Owner: postgres
--

ALTER TABLE ONLY products.product_data
    ADD CONSTRAINT product_data_product_name_key UNIQUE (product_name);


--
-- TOC entry 4915 (class 2606 OID 26779)
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- TOC entry 4958 (class 2606 OID 27017)
-- Name: quotation quotation_pkey; Type: CONSTRAINT; Schema: quotations; Owner: postgres
--

ALTER TABLE ONLY quotations.quotation
    ADD CONSTRAINT quotation_pkey PRIMARY KEY (quotation_id);


--
-- TOC entry 4975 (class 2606 OID 27137)
-- Name: quotations_products quotations_products_pkey; Type: CONSTRAINT; Schema: quotations; Owner: postgres
--

ALTER TABLE ONLY quotations.quotations_products
    ADD CONSTRAINT quotations_products_pkey PRIMARY KEY (quotation_id, product_id);


--
-- TOC entry 4953 (class 2606 OID 26980)
-- Name: recurring_plan_data recurring_plan_data_pkey; Type: CONSTRAINT; Schema: recurring_plans; Owner: postgres
--

ALTER TABLE ONLY recurring_plans.recurring_plan_data
    ADD CONSTRAINT recurring_plan_data_pkey PRIMARY KEY (recurring_plan_id);


--
-- TOC entry 4955 (class 2606 OID 26982)
-- Name: recurring_plan_data recurring_plan_data_recurring_name_key; Type: CONSTRAINT; Schema: recurring_plans; Owner: postgres
--

ALTER TABLE ONLY recurring_plans.recurring_plan_data
    ADD CONSTRAINT recurring_plan_data_recurring_name_key UNIQUE (recurring_name);


--
-- TOC entry 4999 (class 2606 OID 27249)
-- Name: subscription_other_info subscription_other_info_pkey; Type: CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscription_other_info
    ADD CONSTRAINT subscription_other_info_pkey PRIMARY KEY (subscription_other_info_id);


--
-- TOC entry 5001 (class 2606 OID 27251)
-- Name: subscription_other_info subscription_other_info_subscription_id_key; Type: CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscription_other_info
    ADD CONSTRAINT subscription_other_info_subscription_id_key UNIQUE (subscription_id);


--
-- TOC entry 5007 (class 2606 OID 27268)
-- Name: subscription_product_variants subscription_product_variants_pkey; Type: CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscription_product_variants
    ADD CONSTRAINT subscription_product_variants_pkey PRIMARY KEY (subscription_product_variant_id);


--
-- TOC entry 4991 (class 2606 OID 27204)
-- Name: subscription_products subscription_products_pkey; Type: CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscription_products
    ADD CONSTRAINT subscription_products_pkey PRIMARY KEY (subscription_product_id);


--
-- TOC entry 4985 (class 2606 OID 27159)
-- Name: subscriptions subscriptions_pkey; Type: CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscriptions
    ADD CONSTRAINT subscriptions_pkey PRIMARY KEY (subscription_id);


--
-- TOC entry 4987 (class 2606 OID 27161)
-- Name: subscriptions subscriptions_subscription_number_key; Type: CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscriptions
    ADD CONSTRAINT subscriptions_subscription_number_key UNIQUE (subscription_number);


--
-- TOC entry 5009 (class 2606 OID 27270)
-- Name: subscription_product_variants uq_subscription_product_variants_line_attribute; Type: CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscription_product_variants
    ADD CONSTRAINT uq_subscription_product_variants_line_attribute UNIQUE (subscription_product_id, attribute_id);


--
-- TOC entry 5011 (class 2606 OID 27272)
-- Name: subscription_product_variants uq_subscription_product_variants_line_attribute_value; Type: CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscription_product_variants
    ADD CONSTRAINT uq_subscription_product_variants_line_attribute_value UNIQUE (subscription_product_id, attribute_value_id);


--
-- TOC entry 4993 (class 2606 OID 27206)
-- Name: subscription_products uq_subscription_products_subscription_product; Type: CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscription_products
    ADD CONSTRAINT uq_subscription_products_subscription_product UNIQUE (subscription_id, product_id);


--
-- TOC entry 4946 (class 2606 OID 26911)
-- Name: product_tax product_tax_pkey; Type: CONSTRAINT; Schema: taxes; Owner: postgres
--

ALTER TABLE ONLY taxes.product_tax
    ADD CONSTRAINT product_tax_pkey PRIMARY KEY (product_id, tax_id);


--
-- TOC entry 4935 (class 2606 OID 26884)
-- Name: tax_data tax_data_pkey; Type: CONSTRAINT; Schema: taxes; Owner: postgres
--

ALTER TABLE ONLY taxes.tax_data
    ADD CONSTRAINT tax_data_pkey PRIMARY KEY (tax_id);


--
-- TOC entry 4937 (class 2606 OID 26886)
-- Name: tax_data tax_data_tax_name_key; Type: CONSTRAINT; Schema: taxes; Owner: postgres
--

ALTER TABLE ONLY taxes.tax_data
    ADD CONSTRAINT tax_data_tax_name_key UNIQUE (tax_name);


--
-- TOC entry 5023 (class 2606 OID 27360)
-- Name: cart cart_pkey; Type: CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.cart
    ADD CONSTRAINT cart_pkey PRIMARY KEY (cart_item_id);


--
-- TOC entry 5038 (class 2606 OID 27428)
-- Name: password_reset_otp password_reset_otp_pkey; Type: CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.password_reset_otp
    ADD CONSTRAINT password_reset_otp_pkey PRIMARY KEY (password_reset_otp_id);


--
-- TOC entry 5032 (class 2606 OID 27396)
-- Name: payments payments_pkey; Type: CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.payments
    ADD CONSTRAINT payments_pkey PRIMARY KEY (payment_id);


--
-- TOC entry 4919 (class 2606 OID 26838)
-- Name: user user_email_key; Type: CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users."user"
    ADD CONSTRAINT user_email_key UNIQUE (email);


--
-- TOC entry 4921 (class 2606 OID 26836)
-- Name: user user_pkey; Type: CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users."user"
    ADD CONSTRAINT user_pkey PRIMARY KEY (id);


--
-- TOC entry 4926 (class 1259 OID 26870)
-- Name: idx_attributes_attribute_name; Type: INDEX; Schema: attributes; Owner: postgres
--

CREATE INDEX idx_attributes_attribute_name ON attributes.attribute USING btree (attribute_name);


--
-- TOC entry 4929 (class 1259 OID 26871)
-- Name: idx_attributes_attribute_values_attribute_id; Type: INDEX; Schema: attributes; Owner: postgres
--

CREATE INDEX idx_attributes_attribute_values_attribute_id ON attributes.attribute_values USING btree (attribute_id);


--
-- TOC entry 4947 (class 1259 OID 26963)
-- Name: idx_attributes_product_variants_attribute_id; Type: INDEX; Schema: attributes; Owner: postgres
--

CREATE INDEX idx_attributes_product_variants_attribute_id ON attributes.product_variants USING btree (attribute_id);


--
-- TOC entry 4968 (class 1259 OID 27130)
-- Name: idx_discount_is_active; Type: INDEX; Schema: discount; Owner: postgres
--

CREATE INDEX idx_discount_is_active ON discount.discount_data USING btree (is_active);


--
-- TOC entry 4969 (class 1259 OID 27129)
-- Name: idx_discount_name; Type: INDEX; Schema: discount; Owner: postgres
--

CREATE INDEX idx_discount_name ON discount.discount_data USING btree (discount_name);


--
-- TOC entry 4972 (class 1259 OID 27131)
-- Name: idx_discount_products_product_id; Type: INDEX; Schema: discount; Owner: postgres
--

CREATE INDEX idx_discount_products_product_id ON discount.discount_products USING btree (product_id);


--
-- TOC entry 4994 (class 1259 OID 27235)
-- Name: idx_product_discount_discount_id; Type: INDEX; Schema: discount; Owner: postgres
--

CREATE INDEX idx_product_discount_discount_id ON discount.product_discount USING btree (discount_id);


--
-- TOC entry 4959 (class 1259 OID 27090)
-- Name: idx_payment_term_name; Type: INDEX; Schema: payment_term; Owner: postgres
--

CREATE INDEX idx_payment_term_name ON payment_term.payment_term_data USING btree (payment_term_name);


--
-- TOC entry 5012 (class 1259 OID 27317)
-- Name: idx_privileges_role_data_user_id; Type: INDEX; Schema: privileges; Owner: postgres
--

CREATE INDEX idx_privileges_role_data_user_id ON privileges.role_data USING btree (user_id);


--
-- TOC entry 5019 (class 1259 OID 27338)
-- Name: idx_privileges_role_permissions_role_id; Type: INDEX; Schema: privileges; Owner: postgres
--

CREATE INDEX idx_privileges_role_permissions_role_id ON privileges.role_permissions USING btree (role_id);


--
-- TOC entry 4938 (class 1259 OID 27344)
-- Name: idx_product_data_recurring_plan_id; Type: INDEX; Schema: products; Owner: postgres
--

CREATE INDEX idx_product_data_recurring_plan_id ON products.product_data USING btree (recurring_plan_id);


--
-- TOC entry 4939 (class 1259 OID 26905)
-- Name: idx_products_product_data_name; Type: INDEX; Schema: products; Owner: postgres
--

CREATE INDEX idx_products_product_data_name ON products.product_data USING btree (product_name);


--
-- TOC entry 4956 (class 1259 OID 27023)
-- Name: idx_quotation_recurring_plan_id; Type: INDEX; Schema: quotations; Owner: postgres
--

CREATE INDEX idx_quotation_recurring_plan_id ON quotations.quotation USING btree (recurring_plan_id);


--
-- TOC entry 4973 (class 1259 OID 27148)
-- Name: idx_quotations_products_product_id; Type: INDEX; Schema: quotations; Owner: postgres
--

CREATE INDEX idx_quotations_products_product_id ON quotations.quotations_products USING btree (product_id);


--
-- TOC entry 4950 (class 1259 OID 26984)
-- Name: idx_recurring_plan_data_is_active; Type: INDEX; Schema: recurring_plans; Owner: postgres
--

CREATE INDEX idx_recurring_plan_data_is_active ON recurring_plans.recurring_plan_data USING btree (is_active);


--
-- TOC entry 4951 (class 1259 OID 26983)
-- Name: idx_recurring_plan_data_recurring_name; Type: INDEX; Schema: recurring_plans; Owner: postgres
--

CREATE INDEX idx_recurring_plan_data_recurring_name ON recurring_plans.recurring_plan_data USING btree (recurring_name);


--
-- TOC entry 4997 (class 1259 OID 27257)
-- Name: idx_subscription_other_info_subscription_id; Type: INDEX; Schema: subscription; Owner: postgres
--

CREATE INDEX idx_subscription_other_info_subscription_id ON subscription.subscription_other_info USING btree (subscription_id);


--
-- TOC entry 5002 (class 1259 OID 27295)
-- Name: idx_subscription_product_variants_attribute_id; Type: INDEX; Schema: subscription; Owner: postgres
--

CREATE INDEX idx_subscription_product_variants_attribute_id ON subscription.subscription_product_variants USING btree (attribute_id);


--
-- TOC entry 5003 (class 1259 OID 27296)
-- Name: idx_subscription_product_variants_attribute_value_id; Type: INDEX; Schema: subscription; Owner: postgres
--

CREATE INDEX idx_subscription_product_variants_attribute_value_id ON subscription.subscription_product_variants USING btree (attribute_value_id);


--
-- TOC entry 5004 (class 1259 OID 27294)
-- Name: idx_subscription_product_variants_product_id; Type: INDEX; Schema: subscription; Owner: postgres
--

CREATE INDEX idx_subscription_product_variants_product_id ON subscription.subscription_product_variants USING btree (product_id);


--
-- TOC entry 5005 (class 1259 OID 27293)
-- Name: idx_subscription_product_variants_subscription_product_id; Type: INDEX; Schema: subscription; Owner: postgres
--

CREATE INDEX idx_subscription_product_variants_subscription_product_id ON subscription.subscription_product_variants USING btree (subscription_product_id);


--
-- TOC entry 4988 (class 1259 OID 27218)
-- Name: idx_subscription_products_product_id; Type: INDEX; Schema: subscription; Owner: postgres
--

CREATE INDEX idx_subscription_products_product_id ON subscription.subscription_products USING btree (product_id);


--
-- TOC entry 4989 (class 1259 OID 27217)
-- Name: idx_subscription_products_subscription_id; Type: INDEX; Schema: subscription; Owner: postgres
--

CREATE INDEX idx_subscription_products_subscription_id ON subscription.subscription_products USING btree (subscription_id);


--
-- TOC entry 4976 (class 1259 OID 27185)
-- Name: idx_subscriptions_customer_id; Type: INDEX; Schema: subscription; Owner: postgres
--

CREATE INDEX idx_subscriptions_customer_id ON subscription.subscriptions USING btree (customer_id);


--
-- TOC entry 4977 (class 1259 OID 27168)
-- Name: idx_subscriptions_customer_name; Type: INDEX; Schema: subscription; Owner: postgres
--

CREATE INDEX idx_subscriptions_customer_name ON subscription.subscriptions USING btree (customer_name);


--
-- TOC entry 4978 (class 1259 OID 27169)
-- Name: idx_subscriptions_next_invoice_date; Type: INDEX; Schema: subscription; Owner: postgres
--

CREATE INDEX idx_subscriptions_next_invoice_date ON subscription.subscriptions USING btree (next_invoice_date);


--
-- TOC entry 4979 (class 1259 OID 27241)
-- Name: idx_subscriptions_payment_term_id; Type: INDEX; Schema: subscription; Owner: postgres
--

CREATE INDEX idx_subscriptions_payment_term_id ON subscription.subscriptions USING btree (payment_term_id);


--
-- TOC entry 4980 (class 1259 OID 27186)
-- Name: idx_subscriptions_quotation_id; Type: INDEX; Schema: subscription; Owner: postgres
--

CREATE INDEX idx_subscriptions_quotation_id ON subscription.subscriptions USING btree (quotation_id);


--
-- TOC entry 4981 (class 1259 OID 27170)
-- Name: idx_subscriptions_recurring_plan_id; Type: INDEX; Schema: subscription; Owner: postgres
--

CREATE INDEX idx_subscriptions_recurring_plan_id ON subscription.subscriptions USING btree (recurring_plan_id);


--
-- TOC entry 4982 (class 1259 OID 27171)
-- Name: idx_subscriptions_status; Type: INDEX; Schema: subscription; Owner: postgres
--

CREATE INDEX idx_subscriptions_status ON subscription.subscriptions USING btree (status);


--
-- TOC entry 4983 (class 1259 OID 27167)
-- Name: idx_subscriptions_subscription_number; Type: INDEX; Schema: subscription; Owner: postgres
--

CREATE INDEX idx_subscriptions_subscription_number ON subscription.subscriptions USING btree (subscription_number);


--
-- TOC entry 4944 (class 1259 OID 26922)
-- Name: idx_taxes_product_tax_tax_id; Type: INDEX; Schema: taxes; Owner: postgres
--

CREATE INDEX idx_taxes_product_tax_tax_id ON taxes.product_tax USING btree (tax_id);


--
-- TOC entry 4932 (class 1259 OID 26888)
-- Name: idx_taxes_tax_data_computation_unit; Type: INDEX; Schema: taxes; Owner: postgres
--

CREATE INDEX idx_taxes_tax_data_computation_unit ON taxes.tax_data USING btree (tax_computation_unit);


--
-- TOC entry 4933 (class 1259 OID 26887)
-- Name: idx_taxes_tax_data_name; Type: INDEX; Schema: taxes; Owner: postgres
--

CREATE INDEX idx_taxes_tax_data_name ON taxes.tax_data USING btree (tax_name);


--
-- TOC entry 5033 (class 1259 OID 27436)
-- Name: idx_password_reset_otp_created_at; Type: INDEX; Schema: users; Owner: postgres
--

CREATE INDEX idx_password_reset_otp_created_at ON users.password_reset_otp USING btree (created_at DESC);


--
-- TOC entry 5034 (class 1259 OID 27435)
-- Name: idx_password_reset_otp_email; Type: INDEX; Schema: users; Owner: postgres
--

CREATE INDEX idx_password_reset_otp_email ON users.password_reset_otp USING btree (lower((email)::text));


--
-- TOC entry 5035 (class 1259 OID 27437)
-- Name: idx_password_reset_otp_reset_token_hash; Type: INDEX; Schema: users; Owner: postgres
--

CREATE INDEX idx_password_reset_otp_reset_token_hash ON users.password_reset_otp USING btree (reset_token_hash) WHERE (reset_token_hash IS NOT NULL);


--
-- TOC entry 5036 (class 1259 OID 27434)
-- Name: idx_password_reset_otp_user_id; Type: INDEX; Schema: users; Owner: postgres
--

CREATE INDEX idx_password_reset_otp_user_id ON users.password_reset_otp USING btree (user_id);


--
-- TOC entry 5024 (class 1259 OID 27377)
-- Name: idx_users_cart_product_id; Type: INDEX; Schema: users; Owner: postgres
--

CREATE INDEX idx_users_cart_product_id ON users.cart USING btree (product_id);


--
-- TOC entry 5025 (class 1259 OID 27376)
-- Name: idx_users_cart_user_id; Type: INDEX; Schema: users; Owner: postgres
--

CREATE INDEX idx_users_cart_user_id ON users.cart USING btree (user_id);


--
-- TOC entry 5026 (class 1259 OID 27378)
-- Name: idx_users_cart_user_product_variant_unique; Type: INDEX; Schema: users; Owner: postgres
--

CREATE UNIQUE INDEX idx_users_cart_user_product_variant_unique ON users.cart USING btree (user_id, product_id, COALESCE(selected_variant_attribute_id, '00000000-0000-0000-0000-000000000000'::uuid));


--
-- TOC entry 5027 (class 1259 OID 27409)
-- Name: idx_users_payments_payment_date; Type: INDEX; Schema: users; Owner: postgres
--

CREATE INDEX idx_users_payments_payment_date ON users.payments USING btree (payment_date DESC);


--
-- TOC entry 5028 (class 1259 OID 27410)
-- Name: idx_users_payments_paypal_payment_id; Type: INDEX; Schema: users; Owner: postgres
--

CREATE INDEX idx_users_payments_paypal_payment_id ON users.payments USING btree (paypal_payment_id);


--
-- TOC entry 5029 (class 1259 OID 27408)
-- Name: idx_users_payments_subscription_id; Type: INDEX; Schema: users; Owner: postgres
--

CREATE INDEX idx_users_payments_subscription_id ON users.payments USING btree (subscription_id);


--
-- TOC entry 5030 (class 1259 OID 27407)
-- Name: idx_users_payments_user_id; Type: INDEX; Schema: users; Owner: postgres
--

CREATE INDEX idx_users_payments_user_id ON users.payments USING btree (user_id);


--
-- TOC entry 4916 (class 1259 OID 26839)
-- Name: idx_users_user_email; Type: INDEX; Schema: users; Owner: postgres
--

CREATE INDEX idx_users_user_email ON users."user" USING btree (email);


--
-- TOC entry 4917 (class 1259 OID 27863)
-- Name: idx_users_user_role; Type: INDEX; Schema: users; Owner: postgres
--

CREATE INDEX idx_users_user_role ON users."user" USING btree (role);


--
-- TOC entry 5039 (class 2606 OID 26865)
-- Name: attribute_values fk_attribute_values_attribute; Type: FK CONSTRAINT; Schema: attributes; Owner: postgres
--

ALTER TABLE ONLY attributes.attribute_values
    ADD CONSTRAINT fk_attribute_values_attribute FOREIGN KEY (attribute_id) REFERENCES attributes.attribute(attribute_id) ON DELETE CASCADE;


--
-- TOC entry 5043 (class 2606 OID 26958)
-- Name: product_variants fk_product_variants_v2_attribute; Type: FK CONSTRAINT; Schema: attributes; Owner: postgres
--

ALTER TABLE ONLY attributes.product_variants
    ADD CONSTRAINT fk_product_variants_v2_attribute FOREIGN KEY (attribute_id) REFERENCES attributes.attribute(attribute_id) ON DELETE RESTRICT;


--
-- TOC entry 5044 (class 2606 OID 26953)
-- Name: product_variants fk_product_variants_v2_product; Type: FK CONSTRAINT; Schema: attributes; Owner: postgres
--

ALTER TABLE ONLY attributes.product_variants
    ADD CONSTRAINT fk_product_variants_v2_product FOREIGN KEY (product_id) REFERENCES products.product_data(product_id) ON DELETE CASCADE;


--
-- TOC entry 5046 (class 2606 OID 27119)
-- Name: discount_products fk_discount_products_discount; Type: FK CONSTRAINT; Schema: discount; Owner: postgres
--

ALTER TABLE ONLY discount.discount_products
    ADD CONSTRAINT fk_discount_products_discount FOREIGN KEY (discount_id) REFERENCES discount.discount_data(discount_id) ON DELETE CASCADE;


--
-- TOC entry 5047 (class 2606 OID 27124)
-- Name: discount_products fk_discount_products_product; Type: FK CONSTRAINT; Schema: discount; Owner: postgres
--

ALTER TABLE ONLY discount.discount_products
    ADD CONSTRAINT fk_discount_products_product FOREIGN KEY (product_id) REFERENCES products.product_data(product_id) ON DELETE RESTRICT;


--
-- TOC entry 5056 (class 2606 OID 27230)
-- Name: product_discount fk_product_discount_discount; Type: FK CONSTRAINT; Schema: discount; Owner: postgres
--

ALTER TABLE ONLY discount.product_discount
    ADD CONSTRAINT fk_product_discount_discount FOREIGN KEY (discount_id) REFERENCES discount.discount_data(discount_id) ON DELETE RESTRICT;


--
-- TOC entry 5057 (class 2606 OID 27225)
-- Name: product_discount fk_product_discount_product; Type: FK CONSTRAINT; Schema: discount; Owner: postgres
--

ALTER TABLE ONLY discount.product_discount
    ADD CONSTRAINT fk_product_discount_product FOREIGN KEY (product_id) REFERENCES products.product_data(product_id) ON DELETE CASCADE;


--
-- TOC entry 5064 (class 2606 OID 27333)
-- Name: role_permissions fk_privileges_permissions_role; Type: FK CONSTRAINT; Schema: privileges; Owner: postgres
--

ALTER TABLE ONLY privileges.role_permissions
    ADD CONSTRAINT fk_privileges_permissions_role FOREIGN KEY (role_id) REFERENCES privileges.role_data(role_id) ON DELETE CASCADE;


--
-- TOC entry 5063 (class 2606 OID 27312)
-- Name: role_data fk_privileges_role_user; Type: FK CONSTRAINT; Schema: privileges; Owner: postgres
--

ALTER TABLE ONLY privileges.role_data
    ADD CONSTRAINT fk_privileges_role_user FOREIGN KEY (user_id) REFERENCES users."user"(id) ON DELETE CASCADE;


--
-- TOC entry 5040 (class 2606 OID 27339)
-- Name: product_data fk_product_data_recurring_plan; Type: FK CONSTRAINT; Schema: products; Owner: postgres
--

ALTER TABLE ONLY products.product_data
    ADD CONSTRAINT fk_product_data_recurring_plan FOREIGN KEY (recurring_plan_id) REFERENCES recurring_plans.recurring_plan_data(recurring_plan_id) ON DELETE SET NULL;


--
-- TOC entry 5045 (class 2606 OID 27018)
-- Name: quotation fk_quotation_recurring_plan; Type: FK CONSTRAINT; Schema: quotations; Owner: postgres
--

ALTER TABLE ONLY quotations.quotation
    ADD CONSTRAINT fk_quotation_recurring_plan FOREIGN KEY (recurring_plan_id) REFERENCES recurring_plans.recurring_plan_data(recurring_plan_id) ON DELETE RESTRICT;


--
-- TOC entry 5048 (class 2606 OID 27143)
-- Name: quotations_products fk_quotations_products_product; Type: FK CONSTRAINT; Schema: quotations; Owner: postgres
--

ALTER TABLE ONLY quotations.quotations_products
    ADD CONSTRAINT fk_quotations_products_product FOREIGN KEY (product_id) REFERENCES products.product_data(product_id) ON DELETE RESTRICT;


--
-- TOC entry 5049 (class 2606 OID 27138)
-- Name: quotations_products fk_quotations_products_quotation; Type: FK CONSTRAINT; Schema: quotations; Owner: postgres
--

ALTER TABLE ONLY quotations.quotations_products
    ADD CONSTRAINT fk_quotations_products_quotation FOREIGN KEY (quotation_id) REFERENCES quotations.quotation(quotation_id) ON DELETE CASCADE;


--
-- TOC entry 5058 (class 2606 OID 27252)
-- Name: subscription_other_info fk_subscription_other_info_subscription; Type: FK CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscription_other_info
    ADD CONSTRAINT fk_subscription_other_info_subscription FOREIGN KEY (subscription_id) REFERENCES subscription.subscriptions(subscription_id) ON DELETE CASCADE;


--
-- TOC entry 5059 (class 2606 OID 27283)
-- Name: subscription_product_variants fk_subscription_product_variants_attribute; Type: FK CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscription_product_variants
    ADD CONSTRAINT fk_subscription_product_variants_attribute FOREIGN KEY (attribute_id) REFERENCES attributes.attribute(attribute_id) ON DELETE RESTRICT;


--
-- TOC entry 5060 (class 2606 OID 27288)
-- Name: subscription_product_variants fk_subscription_product_variants_attribute_value; Type: FK CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscription_product_variants
    ADD CONSTRAINT fk_subscription_product_variants_attribute_value FOREIGN KEY (attribute_value_id) REFERENCES attributes.attribute_values(attribute_value_id) ON DELETE RESTRICT;


--
-- TOC entry 5061 (class 2606 OID 27278)
-- Name: subscription_product_variants fk_subscription_product_variants_product; Type: FK CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscription_product_variants
    ADD CONSTRAINT fk_subscription_product_variants_product FOREIGN KEY (product_id) REFERENCES products.product_data(product_id) ON DELETE RESTRICT;


--
-- TOC entry 5062 (class 2606 OID 27273)
-- Name: subscription_product_variants fk_subscription_product_variants_subscription_product; Type: FK CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscription_product_variants
    ADD CONSTRAINT fk_subscription_product_variants_subscription_product FOREIGN KEY (subscription_product_id) REFERENCES subscription.subscription_products(subscription_product_id) ON DELETE CASCADE;


--
-- TOC entry 5054 (class 2606 OID 27212)
-- Name: subscription_products fk_subscription_products_product; Type: FK CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscription_products
    ADD CONSTRAINT fk_subscription_products_product FOREIGN KEY (product_id) REFERENCES products.product_data(product_id) ON DELETE RESTRICT;


--
-- TOC entry 5055 (class 2606 OID 27207)
-- Name: subscription_products fk_subscription_products_subscription; Type: FK CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscription_products
    ADD CONSTRAINT fk_subscription_products_subscription FOREIGN KEY (subscription_id) REFERENCES subscription.subscriptions(subscription_id) ON DELETE CASCADE;


--
-- TOC entry 5050 (class 2606 OID 27175)
-- Name: subscriptions fk_subscriptions_customer; Type: FK CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscriptions
    ADD CONSTRAINT fk_subscriptions_customer FOREIGN KEY (customer_id) REFERENCES users."user"(id) ON DELETE RESTRICT;


--
-- TOC entry 5051 (class 2606 OID 27236)
-- Name: subscriptions fk_subscriptions_payment_term; Type: FK CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscriptions
    ADD CONSTRAINT fk_subscriptions_payment_term FOREIGN KEY (payment_term_id) REFERENCES payment_term.payment_term_data(payment_term_id) ON DELETE SET NULL;


--
-- TOC entry 5052 (class 2606 OID 27180)
-- Name: subscriptions fk_subscriptions_quotation; Type: FK CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscriptions
    ADD CONSTRAINT fk_subscriptions_quotation FOREIGN KEY (quotation_id) REFERENCES quotations.quotation(quotation_id) ON DELETE SET NULL;


--
-- TOC entry 5053 (class 2606 OID 27411)
-- Name: subscriptions fk_subscriptions_recurring_plan; Type: FK CONSTRAINT; Schema: subscription; Owner: postgres
--

ALTER TABLE ONLY subscription.subscriptions
    ADD CONSTRAINT fk_subscriptions_recurring_plan FOREIGN KEY (recurring_plan_id) REFERENCES recurring_plans.recurring_plan_data(recurring_plan_id) ON DELETE SET NULL;


--
-- TOC entry 5041 (class 2606 OID 26912)
-- Name: product_tax fk_product_tax_product; Type: FK CONSTRAINT; Schema: taxes; Owner: postgres
--

ALTER TABLE ONLY taxes.product_tax
    ADD CONSTRAINT fk_product_tax_product FOREIGN KEY (product_id) REFERENCES products.product_data(product_id) ON DELETE CASCADE;


--
-- TOC entry 5042 (class 2606 OID 26917)
-- Name: product_tax fk_product_tax_tax; Type: FK CONSTRAINT; Schema: taxes; Owner: postgres
--

ALTER TABLE ONLY taxes.product_tax
    ADD CONSTRAINT fk_product_tax_tax FOREIGN KEY (tax_id) REFERENCES taxes.tax_data(tax_id) ON DELETE RESTRICT;


--
-- TOC entry 5069 (class 2606 OID 27429)
-- Name: password_reset_otp fk_password_reset_otp_user; Type: FK CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.password_reset_otp
    ADD CONSTRAINT fk_password_reset_otp_user FOREIGN KEY (user_id) REFERENCES users."user"(id) ON DELETE CASCADE;


--
-- TOC entry 5065 (class 2606 OID 27366)
-- Name: cart fk_users_cart_product; Type: FK CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.cart
    ADD CONSTRAINT fk_users_cart_product FOREIGN KEY (product_id) REFERENCES products.product_data(product_id) ON DELETE CASCADE;


--
-- TOC entry 5066 (class 2606 OID 27361)
-- Name: cart fk_users_cart_user; Type: FK CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.cart
    ADD CONSTRAINT fk_users_cart_user FOREIGN KEY (user_id) REFERENCES users."user"(id) ON DELETE CASCADE;


--
-- TOC entry 5067 (class 2606 OID 27402)
-- Name: payments fk_users_payments_subscription; Type: FK CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.payments
    ADD CONSTRAINT fk_users_payments_subscription FOREIGN KEY (subscription_id) REFERENCES subscription.subscriptions(subscription_id) ON DELETE SET NULL;


--
-- TOC entry 5068 (class 2606 OID 27397)
-- Name: payments fk_users_payments_user; Type: FK CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.payments
    ADD CONSTRAINT fk_users_payments_user FOREIGN KEY (user_id) REFERENCES users."user"(id) ON DELETE CASCADE;


-- Completed on 2026-05-04 18:23:01

--
-- PostgreSQL database dump complete
--

\unrestrict rJxpRw7lM4JiLxrJep5zUdGAeHw9Sxxi6FWUWdg2fZeR4ixyPcEM1qg3eXZxePm

