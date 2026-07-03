#!/usr/bin/env python3
"""
subai login helper — Cloudflare bypass + V2Board login via cloudscraper.

Called as a subprocess by the `subai login` Go command.
Communicates via JSON on stdin/stdout.

Input (JSON, one line):
  {"method":"v2board","base_url":"https://www.xfltd.org","email":"...","password":"..."}

Output (JSON, one line on success):
  {"ok":true,"subscribe_url":"https://...","token":"...","auth_data":"...",
   "user":{"email":"...","plan":"250G-不限时长","used":...,"total":...}}

On error:
  {"ok":false,"error":"message"}
"""

import json, sys, base64, urllib.parse, re, traceback, time


def solve_geua_challenge(scraper, base_url):
    """Solve the GE-UA JS challenge. Returns True if solved."""
    for attempt in range(3):
        try:
            r = scraper.get(base_url + "/", timeout=60)
            m = re.search(r'var nonce = (\d+)', r.text)
            if not m:
                if "安全检查" not in r.text and "ge_ua" not in r.text[:500].lower():
                    return True  # No challenge
                continue

            nonce = int(m.group(1))
            # requests cookie jar already URL-decodes values
            ge_ua_p = scraper.cookies.get("ge_ua_p", "")
            if not ge_ua_p:
                time.sleep(1)
                continue

            # Compute sum (same JS algorithm)
            total = sum(ord(ch) * (nonce + i) 
                       for i, ch in enumerate(ge_ua_p) if ch.isalnum())

            # POST solution
            r2 = scraper.post(base_url + "/",
                             data={"sum": total, "nonce": nonce},
                             headers={"X-GE-UA-Step": "prev",
                                      "Content-Type": "application/x-www-form-urlencoded"},
                             timeout=30)

            if "安全检查" not in r2.text and "ge_ua" not in r2.text[:500].lower():
                return True  # Solved!

            # Check if server returned ok:true
            try:
                resp = r2.json()
                if resp.get("ok") is True:
                    return True
            except (json.JSONDecodeError, ValueError):
                pass

        except Exception:
            time.sleep(2)

    return False


def do_v2board_login(base_url, email, password):
    """Full V2Board login flow with CF bypass."""
    import cloudscraper

    url_clean = base_url.rstrip('/')

    # Try multiple strategies
    strategies = [
        url_clean,
        url_clean.replace("://", "://www.") if "www." not in url_clean else None,
        url_clean.replace("://www.", "://") if "www." in url_clean else None,
    ]
    strategies = [s for s in strategies if s]

    last_error = None

    for url_base in strategies:
        for attempt in range(3):
            scraper = cloudscraper.create_scraper(
                browser={'browser': 'chrome', 'platform': 'linux', 'desktop': True},
                interpreter='js2py'
            )

            try:
                # Try to solve challenge
                challenge_solved = solve_geua_challenge(scraper, url_base)

                if not challenge_solved:
                    # Try direct login anyway (cloudscraper handles some challenges internally)
                    pass

                # POST login
                r_login = scraper.post(
                    url_base + "/api/v1/passport/auth/login",
                    json={"email": email, "password": password},
                    timeout=30
                )

                if r_login.status_code == 200:
                    try:
                        data = r_login.json()
                        if 'data' in data and 'token' in data['data']:
                            return _handle_login_response(r_login, scraper, url_base)
                    except (json.JSONDecodeError, ValueError, KeyError):
                        pass

                # If login returns challenge page, wait and retry
                text_lower = r_login.text.lower() if r_login.text else ""
                if "安全检查" in text_lower or "ge_ua" in text_lower:
                    time.sleep(2)
                    continue

                last_error = f"login returned {r_login.status_code}: {r_login.text[:200]}"

            except Exception as e:
                last_error = str(e)
                time.sleep(1)
                continue

    raise Exception(
        f"failed to login after multiple attempts. Last error: {last_error}\n"
        "Hints:\n"
        "  - Try running from a different IP (e.g. your home network)\n"
        "  - Or manually obtain the subscribe URL from the panel and add it to the config"
    )


def _handle_login_response(r, scraper, base_url):
    """Process login response and fetch subscribe URL + user info."""
    data = r.json()
    token = data['data']['token']
    auth_data = data['data']['auth_data']

    # Fetch user info
    r_info = scraper.get(
        base_url + "/api/v1/user/info",
        headers={"Authorization": auth_data, "Accept": "application/json"},
        timeout=30
    )
    user_data = r_info.json().get('data', {}) if r_info.status_code == 200 else {}

    # Fetch subscribe info
    r_sub = scraper.get(
        base_url + "/api/v1/user/getSubscribe",
        headers={"Authorization": auth_data, "Accept": "application/json"},
        timeout=30
    )

    subscribe_url = ""
    sub_data = {}
    if r_sub.status_code == 200:
        try:
            sub_data = r_sub.json().get('data', {})
            subscribe_url = sub_data.get('subscribe_url', '')
        except (json.JSONDecodeError, ValueError):
            pass

    if not subscribe_url:
        subscribe_url = user_data.get('subscribe_url', '')

    # Merge data
    plan_name = ''
    if isinstance(sub_data.get('plan'), dict):
        plan_name = sub_data['plan'].get('name', '')
    elif isinstance(user_data.get('plan'), dict):
        plan_name = user_data['plan'].get('name', '')

    used = (sub_data.get('u', 0) or 0) + (sub_data.get('d', 0) or 0)
    transfer = sub_data.get('transfer_enable', user_data.get('transfer_enable', 0))
    uuid_val = sub_data.get('uuid', user_data.get('uuid', ''))

    return {
        'subscribe_url': subscribe_url,
        'token': token,
        'auth_data': auth_data,
        'user': {
            'email': user_data.get('email', sub_data.get('email', '')),
            'plan': plan_name,
            'transfer_enable': transfer,
            'used': used,
            'expired_at': user_data.get('expired_at'),
            'balance': user_data.get('balance', 0),
            'uuid': uuid_val,
        }
    }


def main():
    """Read JSON input from stdin, perform login, write JSON result to stdout."""
    try:
        line = sys.stdin.readline()
        if not line:
            raise Exception("no input received")

        params = json.loads(line)
        method = params.get('method', 'v2board')

        if method == 'v2board':
            result = do_v2board_login(
                params['base_url'],
                params['email'],
                params['password']
            )
        else:
            raise Exception(f"unsupported login method: {method}")

        result['ok'] = True
        print(json.dumps(result, ensure_ascii=False, default=str))
        sys.stdout.flush()

    except Exception as e:
        error_result = {'ok': False, 'error': str(e)}
        print(json.dumps(error_result, ensure_ascii=False))
        sys.stdout.flush()
        sys.exit(1)


if __name__ == '__main__':
    main()