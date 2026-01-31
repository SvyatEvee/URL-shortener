const API_BASE = '';

class ApiService {
    async request(endpoint, options = {}) {
        const url = `${API_BASE}${endpoint}`;
        const config = {
            ...options,
            headers: {
                'Content-Type': 'application/json',
                ...options.headers,
            },
            
        };

        if (config.body && typeof config.body === 'object') {
            config.body = JSON.stringify(config.body);
        }
        

        let retryCount = 0;
        const maxRetries = 1;

        while(retryCount <= maxRetries) {
            
            const response = await fetch(url, config);
            
            if (response.ok || (response.status === 204 && config.method === 'DELETE')) {
                const contentLength = response.headers.get('content-length')
                if (contentLength) {
                    return response.json()
                }
                return
            } else if (response.status == 401 && retryCount === 0) {
                retryCount++;
                
                const refreshToken = localStorage.getItem('refreshToken')

                if (!refreshToken) {
                    logoutAndRedirect();
                    return;
                }

                try {
                    const newTokens = await this.refreshAccessToken(refreshToken);
                    config.headers.Authorization = `Bearer ${newTokens.accessToken}`
                    continue
                } catch  {
                    logoutAndRedirect();
                    return;
                }   
            }
            else {
                const error = new Error(`HTTP ${response.status}: ${response.statusText}`);
                error.status = response.status
                error.data = await response.json()
                throw error
            }
        }
    }

    async refreshAccessToken(refreshToken) {
        const updateTokenUrl = `${API_BASE}/auth/updateToken`;
        
        const updateConfig = {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ refreshToken })
        };

        const updateResponse = await fetch(updateTokenUrl, updateConfig);

        if (!updateResponse.ok) {
            throw new Error('Failed to refresh token');
        }

        const responseBody = await updateResponse.json();
    
        if (!responseBody.accessToken || !responseBody.refreshToken) {
            throw new Error('Invalid token response');
        }

        localStorage.setItem('accessToken', responseBody.accessToken);
        localStorage.setItem('refreshToken', responseBody.refreshToken);

        return {
            accessToken: responseBody.accessToken,
            refreshToken: responseBody.refreshToken
        };
    }

    async login(email, password) {
        return this.request('/auth/login', {
            method: 'POST',
            body: { email, password }
        });
    }

    async logout(refreshToken) {
        return this.request('/auth/logout', {
            method: 'POST',
            body: { refreshToken }
        });
    }

    async register(email, password) {
        return this.request('/auth', {
            method: 'POST',
            body: { email, password }
        });
    }

    async getUrls() {
        const accessToken = localStorage.getItem('accessToken');
        return this.request('/url/urls', {
            headers: { Authorization: `Bearer ${accessToken}` }
        });
    }

    async createUrl(url, alias = null) {
        const accessToken = localStorage.getItem('accessToken');
        return this.request('/url', {
            method: 'POST',
            headers: { 'Authorization': `Bearer ${accessToken}` },
            body: { url, alias }
        });
    }

    async updateUrl(urlId, newUrl) {
        const accessToken = localStorage.getItem('accessToken');
        return this.request(`/url`, {
            method: 'PATCH',
            headers: { 'Authorization': `Bearer ${accessToken}` },
            body: { urlId: urlId, newUrl: newUrl }
        });
    }

    async deleteUrl(urlId) {
        const accessToken = localStorage.getItem('accessToken');
        return this.request(`/url`, {
            method: 'DELETE',
            headers: { 'Authorization': `Bearer ${accessToken}` },
            body: { urlId }
        });
    }
}

function logoutAndRedirect() {
    localStorage.removeItem('accessToken');
    localStorage.removeItem('refreshToken');
    window.location.href = 'login.html';
    return
}  


const apiService = new ApiService();