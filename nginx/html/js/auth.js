document.addEventListener('DOMContentLoaded', function() {
    const loginForm = document.getElementById('loginForm');
    const registerForm = document.getElementById('registerForm');

    // Проверка авторизации при загрузке
    if (localStorage.getItem('accessToken') && localStorage.getItem('refreshToken')) {
        window.location.href = 'dashboard.html';
    }

    if (loginForm) {
        loginForm.addEventListener('submit', handleLogin);
    }

    if (registerForm) {
        registerForm.addEventListener('submit', handleRegister);
    }
});

async function handleLogin(e) {
    e.preventDefault();
    
    const email = document.getElementById('email').value;
    const password = document.getElementById('password').value;

    try {
        const data = await apiService.login(email, password);
        localStorage.setItem('accessToken', data.accessToken);
        localStorage.setItem('refreshToken', data.refreshToken);
        window.location.href = 'dashboard.html';
    } catch (error) {
        if (error.data.message) {
            alert('Ошибка входа: ' + error.data.message);
        }
        else {
            alert('Ошибка входа: ' + error.message);
        }
    }
}

async function handleRegister(e) {
    e.preventDefault();
    
    const email = document.getElementById('email').value;
    const password = document.getElementById('password').value;

    try {
        await apiService.register(email, password);
        alert('Регистрация успешна! Теперь войдите.');
        window.location.href = 'login.html';
    } catch (error) {
        if (error.data) {
            alert('Ошибка регистрации: ' + error.data.message);
        }
        else {
            alert('Ошибка регистрации: ' + error.message);
        }
    }
}