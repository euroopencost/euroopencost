# Supabase Setup — EuroOpenCost Auth

## 1. Projekt erstellen

1. Gehe zu https://supabase.com → "New project"
2. Name: `euroopencost`, Region: Frankfurt (EU)
3. Warte bis das Projekt ready ist (~1 Minute)

## 2. Credentials holen

Settings → API → kopiere:
- **Project URL** → `https://xxx.supabase.co`
- **anon public** key → `eyJ...`

## 3. SQL ausführen

Gehe zu **SQL Editor** → "New query" → folgendes einfügen und ausführen:

```sql
-- Profiles Tabelle (Pro-Status pro User)
create table public.profiles (
  id   uuid references auth.users on delete cascade primary key,
  is_pro boolean not null default false,
  created_at timestamptz default now()
);

-- Row Level Security aktivieren
alter table public.profiles enable row level security;

-- User darf eigenes Profil lesen
create policy "Users can read own profile"
  on public.profiles for select
  using (auth.uid() = id);

-- Automatisch Profil anlegen wenn User sich registriert
create or replace function public.handle_new_user()
returns trigger language plpgsql security definer as $$
begin
  insert into public.profiles (id) values (new.id);
  return new;
end;
$$;

create trigger on_auth_user_created
  after insert on auth.users
  for each row execute procedure public.handle_new_user();
```

## 4. Credentials in Code eintragen

In allen 5 Dateien (`site/index.html`, `site/dashboard.html`, `site/audit.html`, `site/policies.html`, `site/treasury.html`) ersetze:

```javascript
const SUPABASE_URL      = 'YOUR_SUPABASE_URL';
const SUPABASE_ANON_KEY = 'YOUR_SUPABASE_ANON_KEY';
```

mit deinen echten Werten:

```javascript
const SUPABASE_URL      = 'https://xxx.supabase.co';
const SUPABASE_ANON_KEY = 'eyJ...';
```

## 5. Email-Bestätigung (optional)

Authentication → Settings → "Confirm email" → kann für Dev deaktiviert werden.

## 6. User auf Pro setzen (Admin)

Wenn ein User sich registriert hat, gehe zu:
**Table Editor → profiles** → finde den User → setze `is_pro = true`

Oder per SQL:
```sql
update public.profiles
set is_pro = true
where id = (
  select id from auth.users where email = 'user@example.com'
);
```

## 7. Commit & Deploy

```bash
git add site/
git commit -m "feat: add Supabase auth"
git push origin master:main
```
