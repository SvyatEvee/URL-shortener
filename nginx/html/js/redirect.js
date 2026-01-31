
async function redirect() {
    const shortCode = window.location.pathname.replace(/^\//, '');
    const accessToken = localStorage.getItem('accessToken');
    
    try {

        console.log(shortCode);
        console.log(`/url/${shortCode}`);

        const response = await fetch(`/url/${shortCode}`, {
            method: 'GET',
            headers: {
                'Authorization': `Bearer ${accessToken || ''}`
            }
            // redirect: 'manual'
        });
        
        console.log('перед проверкой редиректа' + response.status)
        if (response.ok) {
            console.log('пытаемся в редирект')
            const redirectUrl = response.headers.get('Location');
            if (redirectUrl) {
                window.location.href = redirectUrl;
                return;
            }
        }
        
        if (response.status === 401) {
            console.log('401 ошибка, пробуем обновить токен')
            const refreshToken = localStorage.getItem('refreshToken');
            if (refreshToken) {
                try {
                    await apiService.refreshAccessToken(refreshToken);
                    redirect();
                    return;
                } catch (refreshError) {
                    alert('Ошибка перенаправления: время вашей сессии истекло');
                    console.error('Token refresh failed:', refreshError);
                }
            }
            logoutAndRedirect();   
            return;
        }
        
        console.log('до сюда')
        const error = new Error(`HTTP ${response.status}: ${response.statusText}`);
        error.status = response.status
        error.data = await response.json()
        throw error 

    } catch (error) {
        console.log('Попали сюда')
        console.error('Network error:', error);
        if (error.data.message) {
            console.log('1')
            alert('Ошибка перенаправления: ' + error.data.message);
        }
        else {
            console.log('2')
            alert('Ошибка перенаправления: ' + error.message);
        }
        window.location.href = 'login.html';
    }
}

redirect();