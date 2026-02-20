let socket;
let currentLanguage = 'en';
let alphabet = [];

function startGame(lang) {
    currentLanguage = lang;
    document.getElementById('language-screen').classList.add('hidden');
    document.getElementById('game-screen').classList.remove('hidden');

    initWebSocket();
}

function initWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    socket = new WebSocket(`${protocol}//${window.location.host}/ws`);

    socket.onopen = () => {
        socket.send(JSON.stringify({
            type: 'INIT',
            payload: { language: currentLanguage }
        }));
    };

    socket.onmessage = (event) => {
        const msg = JSON.parse(event.data);
        handleMessage(msg);
    };

    socket.onclose = () => {
        console.log('Socket closed. Attempting to reconnect...');
        setTimeout(initWebSocket, 2000);
    };
}

function handleMessage(msg) {
    switch (msg.type) {
        case 'QUESTION':
            updateUI(msg.payload);
            break;
        case 'TIMER_SYNC':
            updateTimer(msg.payload.timeRemaining);
            break;
        case 'FEEDBACK':
            // Feedback is usually followed by a new QUESTION message which updates the whole state
            break;
        case 'GAME_OVER':
            showGameOver(msg.payload);
            break;
    }
}

function updateUI(data) {
    const { letter, question, lettersState, timeRemaining } = data;

    // Initialize circle if not already done
    if (alphabet.length === 0) {
        alphabet = Object.keys(lettersState).sort((a,b) => {
            // This is a bit tricky since map doesn't guarantee order.
            // In a real app we'd get the sorted list of letters from server.
            return 0; // We'll handle order by the sequence they appear if needed
        });
        // Actually, let's just use the letters from the keys and sort them if possible
        // but Turkish sorting is special.
        // For simplicity, let's just use the order the server provides in lettersState
        // IF we could. Since it's a map, we can't.
        // Let's assume the server sends the alphabet in the INIT or we just use a hardcoded one for layout.

        // Better: lettersState is a map. Let's just use the keys and layout them.
        renderCircle(Object.keys(lettersState));
    }

    document.getElementById('current-letter').innerText = letter;
    document.getElementById('question-text').innerText = question;
    document.getElementById('answer-input').value = '';
    document.getElementById('answer-input').focus();
    updateTimer(timeRemaining);

    // Update letter colors
    Object.keys(lettersState).forEach(l => {
        const node = document.getElementById(`letter-${l}`);
        if (node) {
            node.className = 'letter-node';
            if (lettersState[l]) node.classList.add(lettersState[l]);
            if (l === letter) node.classList.add('active');
        }
    });
}

function renderCircle(letters) {
    // Sort letters based on language
    if (currentLanguage === 'tr') {
        const trAlphabet = "ABCÇDEFGĞHIİJKLMNOÖPRSŞTUÜVYZ";
        letters.sort((a, b) => trAlphabet.indexOf(a) - trAlphabet.indexOf(b));
    } else {
        letters.sort();
    }
    alphabet = letters;

    const container = document.getElementById('alphabet-circle');
    container.innerHTML = '';
    const radius = container.offsetWidth / 2;
    const centerX = radius;
    const centerY = radius;
    const angleStep = (2 * Math.PI) / letters.length;

    letters.forEach((l, i) => {
        const angle = i * angleStep - Math.PI / 2;
        const x = centerX + (radius - 30) * Math.cos(angle);
        const y = centerY + (radius - 30) * Math.sin(angle);

        const node = document.createElement('div');
        node.id = `letter-${l}`;
        node.className = 'letter-node';
        node.innerText = l;
        node.style.left = `${x}px`;
        node.style.top = `${y}px`;
        container.appendChild(node);
    });
}

function updateTimer(seconds) {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    document.getElementById('timer-display').innerText =
        `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
}

function submitAnswer() {
    const answer = document.getElementById('answer-input').value;
    if (!answer) return;
    socket.send(JSON.stringify({
        type: 'ANSWER',
        payload: { answer: answer }
    }));
}

function passQuestion() {
    socket.send(JSON.stringify({
        type: 'PASS'
    }));
}

function showGameOver(stats) {
    document.getElementById('game-screen').classList.add('hidden');
    document.getElementById('result-screen').classList.remove('hidden');
    document.getElementById('stat-correct').innerText = stats.correctCount;
    document.getElementById('stat-wrong').innerText = stats.wrongCount;
    document.getElementById('stat-passed').innerText = stats.passedCount;
}

// Handle Enter key in input
document.getElementById('answer-input').addEventListener('keypress', (e) => {
    if (e.key === 'Enter') submitAnswer();
});
