document.addEventListener('DOMContentLoaded', function() {
    console.log('Dashboard.js загружен!');
    const accessToken = localStorage.getItem('accessToken');
    const refreshToken = localStorage.getItem('refreshToken');
    if (!(accessToken && refreshToken)) {
        window.location.href = 'login.html';
        return;
    }

    // Элементы
    const logoutBtn = document.getElementById('logoutBtn');
    const addUrlForm = document.getElementById('addUrlForm');

    // Обработчики
    logoutBtn.addEventListener('click', handleLogout);
    addUrlForm.addEventListener('submit', handleAddUrl);
    
    // Загрузка URL
    loadUrls();
});

async function handleLogout() {
    const accessToken = localStorage.getItem('accessToken')
    const refreshToken = localStorage.getItem('refreshToken')

    if (!accessToken || !refreshToken) {
        localStorage.removeItem('accessToken')
        localStorage.removeItem('refreshToken')
        window.location.href = 'login.html';
        return
    }

    try {
        await apiService.logout(refreshToken)
    }
    catch (error) {
        if (error.data.message) {
            alert("Ошибка: " + error.data.message)
        }   
        else {
            alert('Ошибка: ' + error.message);
        }
    }
    localStorage.removeItem('accessToken');
    localStorage.removeItem('refreshToken');
    window.location.href = 'login.html';
    return
}

async function handleAddUrl(e) {
    e.preventDefault();
    
    const url = document.getElementById('url').value;
    const alias = document.getElementById('alias').value || null;

    try {
        await apiService.createUrl(url, alias);
        document.getElementById('addUrlForm').reset();
        loadUrls(); // Обновляем список
    } catch (error) {
        if (error.data.message) {
            alert('Ошибка создания URL: ' + error.data.message)
        }   
        else {
            alert('Ошибка создания URL: ' + error.message);
        }
    }
}

async function loadUrls() {
    console.log('Загрузка URL...');
    try {
        const urls = await apiService.getUrls();
        console.log('Получены URL:', urls);
        displayUrls(urls);
    } catch (error) {
        if (error.data.message) {
            alert('Ошибка загрузки URL: ' + error.data.message);
        }
        else {
            alert('Ошибка загрузки URL: ' + error.message);
        }
    }
}

function displayUrls(urls) {
    console.log('Отображение URL:', urls);
    const container = document.getElementById('urlsContainer');
    
    if (urls.length === 0) {
        container.innerHTML = '<p>У вас пока нет сокращённых URL</p>';
        return;
    }

    container.innerHTML = urls.map(url => `
        <div class="url-item">
            <div class="url-info">
                <strong>Оригинальный:</strong> 
                <a href="${url.url}" target="_blank">${url.url}</a><br>
                <strong>Сокращённый:</strong> 
                <a href="${url.url}" target="_blank">${url.alias}</a><br>
            </div>
            <div class="url-actions">
                <button onclick="editUrl(${url.id})">Изменить</button>
                <button onclick="deleteUrl(${url.id})">Удалить</button>
            </div>
        </div>
    `).join('');
}

async function deleteUrl(urlId) {
    if (!confirm('Удалить эту ссылку?')) return;
    
    console.log('Удаление URL с ID:', urlId);
    try {
        console.log('Запрос к бекенду')
        await apiService.deleteUrl(urlId);
        console.log('URL удален, загружаем обновленный список...');
        loadUrls(); // Обновляем список
    } catch (error) {
        console.log('Воникла ошибка')

        if (error.data.message) {
            console.log('попали сюда 1')
            alert('Ошибка удаления: ' + error.data.message);
        }
        else {
            console.log('попали сюда 2')
            alert('Ошибка удаления: ' + error.message);
        }
    }
}

async function editUrl(urlId) {
    const newUrl = prompt('Введите новый URL:');
    if (newUrl) {
        try {
            await apiService.updateUrl(urlId, newUrl);
            loadUrls(); // Обновляем список
        } catch (error) {
            if (error.data.message) {
                alert('Ошибка обновления: ' + error.data.message);
            }
            else {
                alert('Ошибка обновления: ' + error.message);
            }
        }
    }
}