#!/usr/bin/env python3
"""
subai login helper — V2Board login.

Two strategies:
1. Playwright (headless Chromium) — primary for sites with JS/geetest challenges
2. cloudscraper (CF bypass, fast) — fallback for simpler sites

Input: JSON over stdin one line
Output: JSON over stdout one line
"""

import json, sys, time


def _retry(fn, *args, max_attempts=3, delay=3):
    """Retry a strategy up to max_attempts times with delay between attempts."""
    for attempt in range(1, max_attempts + 1):
        try:
            return fn(*args)
        except Exception as e:
            if attempt == max_attempts:
                raise
            sys.stderr.write(f"  ⚠️  attempt {attempt}/{max_attempts} failed, retrying in {delay}s...\n")
            time.sleep(delay)


# ---- Strategy 1: Playwright (primary) ----
def try_playwright(base_url, email, password):
    """Use Playwright to solve CF/geetest challenge, then Python requests for API calls."""
    from playwright.sync_api import sync_playwright
    import requests as py_requests
    url_clean = base_url.rstrip('/')

    with sync_playwright() as p:
        browser = p.chromium.launch(
            headless=True,
            args=['--no-sandbox', '--disable-setuid-sandbox'],
        )
        context = browser.new_context(
            user_agent="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
                       "(KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
        )
        page = context.new_page()

        # Navigate and wait for Cloudflare challenge to resolve
        page.goto(url_clean + "/", timeout=30000)
        for _ in range(20):
            try:
                title = page.title()
                if title and '安全检查' not in title and 'Just a moment' not in title:
                    break
            except:
                pass
            page.wait_for_timeout(1000)

        page.wait_for_load_state("networkidle", timeout=15000)
        page.wait_for_timeout(2000)

        # Extract cookies for requests session
        session = py_requests.Session()
        for cookie in context.cookies():
            session.cookies.set(cookie['name'], cookie['value'],
                                domain=cookie['domain'], path=cookie['path'])

        session.headers.update({
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
                          "(KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
        })

        browser.close()

    return _v2board_api(session, url_clean, email, password)


# ---- Strategy 2: cloudscraper ----
def try_cloudscraper(base_url, email, password):
    """Use cloudscraper to bypass Cloudflare and login."""
    import cloudscraper
    url_clean = base_url.rstrip('/')

    scraper = cloudscraper.create_scraper(
        browser={'browser': 'chrome', 'platform': 'windows', 'desktop': True, 'mobile': False},
        delay=10,
    )
    scraper.headers.update({
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
                       "(KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
    })

    # Warm up — solve CF challenge (best-effort)
    r = scraper.get(url_clean + "/", timeout=30)

    # Login
    r = scraper.post(
        url_clean + "/api/v1/passport/auth/login",
        json={"email": email, "password": password},
        headers={
            "Accept": "application/json", "Content-Type": "application/json",
            "Origin": url_clean, "Referer": url_clean + "/",
            "X-Requested-With": "XMLHttpRequest",
        },
        timeout=15,
    )
    return _parse_login_response(r, url_clean, scraper)


# ---- Shared API helpers ----
def _v2board_api(session, url_clean, email, password):
    """Login to V2Board and fetch user info + subscribe URL via requests session."""
    r = session.post(
        url_clean + "/api/v1/passport/auth/login",
        json={"email": email, "password": password},
        headers={
            "Accept": "application/json", "Content-Type": "application/json",
            "Origin": url_clean, "Referer": url_clean + "/",
            "X-Requested-With": "XMLHttpRequest",
        },
        timeout=15,
    )

    data = r.json()
    if 'data' not in data or 'token' not in data['data']:
        raise Exception(f"login failed: {r.text[:200]}")

    token = data['data']['token']
    auth_data = data['data']['auth_data']

    r_info = session.get(url_clean + "/api/v1/user/info",
        headers={"Authorization": auth_data, "Accept": "application/json"}, timeout=10)
    user_data = r_info.json().get('data', {}) if r_info.status_code == 200 else {}

    r_sub = session.get(url_clean + "/api/v1/user/getSubscribe",
        headers={"Authorization": auth_data, "Accept": "application/json"}, timeout=10)
    sub_data = r_sub.json().get('data', {}) if r_sub.status_code == 200 else {}

    subscribe_url = sub_data.get('subscribe_url', '') or user_data.get('subscribe_url', '')
    plan_name = ''
    if isinstance(sub_data.get('plan'), dict):
        plan_name = sub_data['plan'].get('name', '')
    elif isinstance(user_data.get('plan'), dict):
        plan_name = user_data['plan'].get('name', '')

    used = (sub_data.get('u', 0) or 0) + (sub_data.get('d', 0) or 0)
    transfer = sub_data.get('transfer_enable', user_data.get('transfer_enable', 0))

    return {
        'subscribe_url': subscribe_url, 'token': token, 'auth_data': auth_data,
        'user': {
            'email': user_data.get('email', sub_data.get('email', '')),
            'plan': plan_name, 'transfer_enable': transfer, 'used': used,
            'expired_at': user_data.get('expired_at'), 'balance': user_data.get('balance', 0),
        }
    }


def _parse_login_response(r, url_clean, session):
    """Parse login response and fetch user info + subscribe URL."""
    if r.status_code != 200 or not r.text:
        raise Exception(f"login returned {r.status_code}: empty body")
    data = r.json()
    if 'data' not in data or 'token' not in data['data']:
        raise Exception(f"unexpected login response: {r.text[:200]}")

    token = data['data']['token']
    auth_data = data['data']['auth_data']

    r_info = session.get(url_clean + "/api/v1/user/info",
        headers={"Authorization": auth_data, "Accept": "application/json"}, timeout=10)
    user_data = r_info.json().get('data', {}) if r_info.status_code == 200 else {}

    r_sub = session.get(url_clean + "/api/v1/user/getSubscribe",
        headers={"Authorization": auth_data, "Accept": "application/json"}, timeout=10)
    sub_data = r_sub.json().get('data', {}) if r_sub.status_code == 200 else {}

    subscribe_url = sub_data.get('subscribe_url', '') or user_data.get('subscribe_url', '')
    plan_name = ''
    if isinstance(sub_data.get('plan'), dict):
        plan_name = sub_data['plan'].get('name', '')
    elif isinstance(user_data.get('plan'), dict):
        plan_name = user_data['plan'].get('name', '')

    used = (sub_data.get('u', 0) or 0) + (sub_data.get('d', 0) or 0)
    transfer = sub_data.get('transfer_enable', user_data.get('transfer_enable', 0))

    return {
        'subscribe_url': subscribe_url, 'token': token, 'auth_data': auth_data,
        'user': {
            'email': user_data.get('email', sub_data.get('email', '')),
            'plan': plan_name, 'transfer_enable': transfer, 'used': used,
            'expired_at': user_data.get('expired_at'), 'balance': user_data.get('balance', 0),
        }
    }


def main():
    try:
        line = sys.stdin.readline()
        if not line:
            raise Exception("no input received")
        params = json.loads(line)
        method = params.get('method', 'v2board')
        if method != 'v2board':
            raise Exception(f"unsupported login method: {method}")

        base_url = params['base_url']
        email = params['email']
        password = params['password']

        strategies = [
            ("Playwright", try_playwright),
            ("cloudscraper", try_cloudscraper),
        ]

        last_error = None
        for name, strategy in strategies:
            try:
                result = _retry(strategy, base_url, email, password)
                result['ok'] = True
                print(json.dumps(result, ensure_ascii=False, default=str))
                sys.stdout.flush()
                return
            except ImportError:
                sys.stderr.write(f"  ⚠️  {name} not available, skipping...\n")
                continue
            except Exception as e:
                sys.stderr.write(f"  ⚠️  {name} failed ({e}), trying next...\n")
                last_error = str(e)
                continue

        raise Exception(f"all login strategies failed. Last error: {last_error}")

    except Exception as e:
        print(json.dumps({'ok': False, 'error': str(e)}, ensure_ascii=False), flush=True)
        sys.exit(1)


if __name__ == '__main__':
    main()