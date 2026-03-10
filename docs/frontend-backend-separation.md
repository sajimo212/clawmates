# Frontend Migration Checklist (Next.js -> Gateway)

Current frontend still uses Supabase client directly in many pages.
To complete strict frontend/backend separation, migrate in this order.

## 1. Add gateway base URL

- Frontend env:
  - `NEXT_PUBLIC_API_BASE_URL=http://localhost:8080`

## 2. Replace direct Supabase DB calls in pages

Target files:
- `src/app/dashboard/page.tsx`
- `src/app/dashboard/agent/page.tsx`
- `src/app/dashboard/network/page.tsx`
- `src/app/dashboard/directives/page.tsx`
- `src/app/dashboard/reports/page.tsx`

Replace `supabase.from(...).select/insert/update` with `fetch(${NEXT_PUBLIC_API_BASE_URL}/...)`.

## 3. Auth strategy decision

Choose one and implement consistently:

1. Keep Supabase Auth only for login/session, gateway trusts JWT and maps `user_id`.
2. Move all auth to gateway (e.g. JWT issued by backend).

## 4. Remove Next.js API routes (optional)

After frontend fully talks to gateway, these can be removed:
- `src/app/api/**`

## 5. CORS and deployment

- Enable CORS on gateway for frontend domain.
- Deploy gateway/core/matching separately from frontend.
