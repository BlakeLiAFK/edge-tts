// Edge TTS Studio - 前端逻辑

(function() {
    'use strict';

    // 状态管理
    const state = {
        languages: [],
        currentLanguage: null,
        currentVoice: null,
        isLoading: false,
        history: []
    };

    // DOM 元素
    const elements = {
        languageSelect: document.getElementById('language-select'),
        voiceSelect: document.getElementById('voice-select'),
        sampleBtn: document.getElementById('sample-btn'),
        textInput: document.getElementById('text-input'),
        charCount: document.getElementById('char-count'),
        rateSlider: document.getElementById('rate-slider'),
        rateValue: document.getElementById('rate-value'),
        pitchSlider: document.getElementById('pitch-slider'),
        pitchValue: document.getElementById('pitch-value'),
        srtCheckbox: document.getElementById('srt-checkbox'),
        previewBtn: document.getElementById('preview-btn'),
        downloadBtn: document.getElementById('download-btn'),
        playerSection: document.getElementById('player-section'),
        audioPlayer: document.getElementById('audio-player'),
        loadingOverlay: document.getElementById('loading-overlay'),
        loadingText: document.getElementById('loading-text'),
        historyList: document.getElementById('history-list'),
        clearHistoryBtn: document.getElementById('clear-history-btn')
    };

    // 初始化
    async function init() {
        loadHistory();
        await loadVoices();
        bindEvents();
    }

    // 加载语音列表
    async function loadVoices() {
        try {
            const response = await fetch('/api/voices');
            const data = await response.json();
            state.languages = data.languages || [];
            renderLanguageOptions();
        } catch (error) {
            console.error('Failed to load voices:', error);
            elements.languageSelect.innerHTML = '<option value="">加载失败，请刷新重试</option>';
        }
    }

    // 渲染语言选项
    function renderLanguageOptions() {
        const options = state.languages.map(lang =>
            `<option value="${lang.code}">${lang.name} (${lang.voices.length})</option>`
        ).join('');
        elements.languageSelect.innerHTML = '<option value="">请选择语言</option>' + options;

        // 默认选中中文（简体）
        const defaultLang = state.languages.find(l => l.code === 'zh-CN');
        if (defaultLang) {
            elements.languageSelect.value = 'zh-CN';
            onLanguageChange();
        }
    }

    // 语言变更处理
    function onLanguageChange() {
        const langCode = elements.languageSelect.value;
        state.currentLanguage = state.languages.find(l => l.code === langCode);

        if (!state.currentLanguage) {
            elements.voiceSelect.innerHTML = '<option value="">请先选择语言</option>';
            state.currentVoice = null;
            return;
        }

        const voices = state.currentLanguage.voices;
        const options = voices.map(v => {
            const gender = v.gender === 'Female' ? '女' : '男';
            const styles = v.styles && v.styles.length > 0 ? ` - ${v.styles.slice(0, 2).join('/')}` : '';
            return `<option value="${v.id}">${v.name} (${gender})${styles}</option>`;
        }).join('');

        elements.voiceSelect.innerHTML = options;
        state.currentVoice = voices[0];
    }

    // 音色变更处理
    function onVoiceChange() {
        const voiceId = elements.voiceSelect.value;
        if (state.currentLanguage) {
            state.currentVoice = state.currentLanguage.voices.find(v => v.id === voiceId);
        }
    }

    // 试听音色
    async function playSample() {
        const voiceId = elements.voiceSelect.value;
        if (!voiceId) {
            alert('请先选择音色');
            return;
        }

        try {
            elements.sampleBtn.disabled = true;
            elements.sampleBtn.innerHTML = '<span class="icon">...</span>';

            const response = await fetch(`/api/voices/${encodeURIComponent(voiceId)}/sample?t=${Date.now()}`);
            if (!response.ok) {
                throw new Error('获取音频失败');
            }

            const arrayBuffer = await response.arrayBuffer();
            const blob = new Blob([arrayBuffer], { type: 'audio/mpeg' });
            const audioUrl = URL.createObjectURL(blob);

            // 先设置事件监听器，再设置 src
            const loadPromise = new Promise((resolve, reject) => {
                const onCanPlay = () => {
                    elements.audioPlayer.removeEventListener('canplaythrough', onCanPlay);
                    elements.audioPlayer.removeEventListener('error', onError);
                    resolve();
                };
                const onError = (e) => {
                    elements.audioPlayer.removeEventListener('canplaythrough', onCanPlay);
                    elements.audioPlayer.removeEventListener('error', onError);
                    console.error('Audio error:', e);
                    reject(new Error('音频加载失败'));
                };
                elements.audioPlayer.addEventListener('canplaythrough', onCanPlay);
                elements.audioPlayer.addEventListener('error', onError);
            });

            elements.audioPlayer.src = audioUrl;
            elements.playerSection.style.display = 'block';

            await loadPromise;
            elements.audioPlayer.play().catch(e => console.log('Auto-play blocked:', e));
        } catch (error) {
            console.error('Sample playback failed:', error);
            alert('试听失败，请重试');
        } finally {
            elements.sampleBtn.disabled = false;
            elements.sampleBtn.innerHTML = '<span class="icon">&#9834;</span>';
        }
    }

    // 更新字数统计
    function updateCharCount() {
        const count = elements.textInput.value.length;
        elements.charCount.textContent = count;
    }

    // 格式化语速值
    function formatRateValue(value) {
        if (value === 0) return '正常';
        return value > 0 ? `+${value}%` : `${value}%`;
    }

    // 格式化音调值
    function formatPitchValue(value) {
        if (value === 0) return '正常';
        return value > 0 ? `+${value}Hz` : `${value}Hz`;
    }

    // 更新语速显示
    function updateRateValue() {
        const value = parseInt(elements.rateSlider.value);
        elements.rateValue.textContent = formatRateValue(value);
    }

    // 更新音调显示
    function updatePitchValue() {
        const value = parseInt(elements.pitchSlider.value);
        elements.pitchValue.textContent = formatPitchValue(value);
    }

    // 显示加载状态
    function showLoading(text) {
        elements.loadingText.textContent = text;
        elements.loadingOverlay.style.display = 'flex';
        state.isLoading = true;
    }

    // 隐藏加载状态
    function hideLoading() {
        elements.loadingOverlay.style.display = 'none';
        state.isLoading = false;
    }

    // 获取当前参数
    function getCurrentParams() {
        const rate = parseInt(elements.rateSlider.value);
        const pitch = parseInt(elements.pitchSlider.value);
        return {
            text: elements.textInput.value.trim(),
            voice: elements.voiceSelect.value,
            rate: rate === 0 ? '+0%' : (rate > 0 ? `+${rate}%` : `${rate}%`),
            pitch: pitch === 0 ? '+0Hz' : (pitch > 0 ? `+${pitch}Hz` : `${pitch}Hz`),
            withSrt: elements.srtCheckbox.checked
        };
    }

    // 预览
    async function preview() {
        const params = getCurrentParams();

        if (!params.text) {
            alert('请输入要转换的文本');
            elements.textInput.focus();
            return;
        }

        if (!params.voice) {
            alert('请选择音色');
            return;
        }

        try {
            showLoading('正在生成预览...');

            const response = await fetch('/api/preview', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    text: params.text,
                    voice: params.voice,
                    rate: params.rate,
                    pitch: params.pitch
                })
            });

            if (!response.ok) {
                throw new Error(await response.text());
            }

            const arrayBuffer = await response.arrayBuffer();
            const blob = new Blob([arrayBuffer], { type: 'audio/mpeg' });
            const audioUrl = URL.createObjectURL(blob);

            // 先设置事件监听器，再设置 src
            const loadPromise = new Promise((resolve, reject) => {
                const onCanPlay = () => {
                    elements.audioPlayer.removeEventListener('canplaythrough', onCanPlay);
                    elements.audioPlayer.removeEventListener('error', onError);
                    resolve();
                };
                const onError = (e) => {
                    elements.audioPlayer.removeEventListener('canplaythrough', onCanPlay);
                    elements.audioPlayer.removeEventListener('error', onError);
                    console.error('Audio error:', e);
                    reject(new Error('音频加载失败'));
                };
                elements.audioPlayer.addEventListener('canplaythrough', onCanPlay);
                elements.audioPlayer.addEventListener('error', onError);
            });

            elements.audioPlayer.src = audioUrl;
            elements.playerSection.style.display = 'block';
            elements.playerSection.classList.add('fade-in');

            await loadPromise;
            elements.audioPlayer.play().catch(e => console.log('Auto-play blocked:', e));

        } catch (error) {
            console.error('Preview failed:', error);
            alert('预览失败: ' + error.message);
        } finally {
            hideLoading();
        }
    }

    // 下载
    async function download() {
        const params = getCurrentParams();

        if (!params.text) {
            alert('请输入要转换的文本');
            elements.textInput.focus();
            return;
        }

        if (!params.voice) {
            alert('请选择音色');
            return;
        }

        try {
            showLoading(params.withSrt ? '正在生成音频和字幕...' : '正在生成音频...');

            const response = await fetch('/api/synthesize', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    text: params.text,
                    voice: params.voice,
                    rate: params.rate,
                    pitch: params.pitch,
                    withSrt: params.withSrt
                })
            });

            if (!response.ok) {
                throw new Error(await response.text());
            }

            const timestamp = Date.now();

            if (params.withSrt) {
                // 返回 JSON，包含 base64 音频和 SRT
                const data = await response.json();

                // 下载音频
                const audioBlob = base64ToBlob(data.audio, 'audio/mpeg');
                downloadBlob(audioBlob, `tts_${timestamp}.mp3`);

                // 下载字幕
                if (data.srt) {
                    const srtBlob = new Blob([data.srt], { type: 'text/plain' });
                    downloadBlob(srtBlob, `tts_${timestamp}.srt`);
                }
            } else {
                // 直接下载音频
                const blob = await response.blob();
                downloadBlob(blob, `tts_${timestamp}.mp3`);
            }

            // 添加到历史记录
            addToHistory({
                text: params.text,
                voice: params.voice,
                voiceName: state.currentVoice?.name || params.voice,
                rate: params.rate,
                pitch: params.pitch,
                timestamp: timestamp
            });

        } catch (error) {
            console.error('Download failed:', error);
            alert('下载失败: ' + error.message);
        } finally {
            hideLoading();
        }
    }

    // Base64 转 Blob
    function base64ToBlob(base64, mimeType) {
        const byteString = atob(base64);
        const ab = new ArrayBuffer(byteString.length);
        const ia = new Uint8Array(ab);
        for (let i = 0; i < byteString.length; i++) {
            ia[i] = byteString.charCodeAt(i);
        }
        return new Blob([ab], { type: mimeType });
    }

    // 下载 Blob
    function downloadBlob(blob, filename) {
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    }

    // 历史记录相关
    function loadHistory() {
        try {
            const saved = localStorage.getItem('tts_history');
            if (saved) {
                state.history = JSON.parse(saved);
                renderHistory();
            }
        } catch (e) {
            state.history = [];
        }
    }

    function saveHistory() {
        try {
            // 只保留最近 20 条
            const toSave = state.history.slice(0, 20);
            localStorage.setItem('tts_history', JSON.stringify(toSave));
        } catch (e) {
            console.error('Failed to save history:', e);
        }
    }

    function addToHistory(item) {
        state.history.unshift(item);
        saveHistory();
        renderHistory();
    }

    function clearHistory() {
        if (confirm('确定要清空所有历史记录吗？')) {
            state.history = [];
            saveHistory();
            renderHistory();
        }
    }

    function renderHistory() {
        if (state.history.length === 0) {
            elements.historyList.innerHTML = '<p class="history-empty">暂无历史记录</p>';
            return;
        }

        const html = state.history.map((item, index) => {
            const date = new Date(item.timestamp);
            const timeStr = `${date.getMonth() + 1}/${date.getDate()} ${date.getHours()}:${String(date.getMinutes()).padStart(2, '0')}`;
            const textPreview = item.text.length > 30 ? item.text.substring(0, 30) + '...' : item.text;

            return `
                <div class="history-item fade-in" data-index="${index}">
                    <div class="history-item-icon">&#9834;</div>
                    <div class="history-item-content">
                        <div class="history-item-text">${escapeHtml(textPreview)}</div>
                        <div class="history-item-meta">${item.voiceName} | ${timeStr}</div>
                    </div>
                    <div class="history-item-actions">
                        <button class="history-item-btn history-load-btn" data-index="${index}" title="加载">&#8634;</button>
                        <button class="history-item-btn history-delete-btn" data-index="${index}" title="删除">&#10005;</button>
                    </div>
                </div>
            `;
        }).join('');

        elements.historyList.innerHTML = html;

        // 绑定历史记录项事件
        elements.historyList.querySelectorAll('.history-load-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.stopPropagation();
                loadHistoryItem(parseInt(btn.dataset.index));
            });
        });

        elements.historyList.querySelectorAll('.history-delete-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.stopPropagation();
                deleteHistoryItem(parseInt(btn.dataset.index));
            });
        });
    }

    function loadHistoryItem(index) {
        const item = state.history[index];
        if (!item) return;

        // 填充文本
        elements.textInput.value = item.text;
        updateCharCount();

        // 尝试选择语音
        const voiceId = item.voice;
        const locale = voiceId.split('-').slice(0, 2).join('-');

        // 选择语言
        if (elements.languageSelect.value !== locale) {
            elements.languageSelect.value = locale;
            onLanguageChange();
        }

        // 选择音色
        elements.voiceSelect.value = voiceId;
        onVoiceChange();

        // 设置参数
        const rateMatch = item.rate.match(/([+-]?\d+)/);
        if (rateMatch) {
            elements.rateSlider.value = parseInt(rateMatch[1]);
            updateRateValue();
        }

        const pitchMatch = item.pitch.match(/([+-]?\d+)/);
        if (pitchMatch) {
            elements.pitchSlider.value = parseInt(pitchMatch[1]);
            updatePitchValue();
        }

        // 滚动到顶部
        window.scrollTo({ top: 0, behavior: 'smooth' });
    }

    function deleteHistoryItem(index) {
        state.history.splice(index, 1);
        saveHistory();
        renderHistory();
    }

    // HTML 转义
    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // 绑定事件
    function bindEvents() {
        elements.languageSelect.addEventListener('change', onLanguageChange);
        elements.voiceSelect.addEventListener('change', onVoiceChange);
        elements.sampleBtn.addEventListener('click', playSample);
        elements.textInput.addEventListener('input', updateCharCount);
        elements.rateSlider.addEventListener('input', updateRateValue);
        elements.pitchSlider.addEventListener('input', updatePitchValue);
        elements.previewBtn.addEventListener('click', preview);
        elements.downloadBtn.addEventListener('click', download);
        elements.clearHistoryBtn.addEventListener('click', clearHistory);

        // 键盘快捷键
        document.addEventListener('keydown', (e) => {
            if (e.ctrlKey || e.metaKey) {
                if (e.key === 'Enter' && !state.isLoading) {
                    e.preventDefault();
                    preview();
                } else if (e.key === 's' && !state.isLoading) {
                    e.preventDefault();
                    download();
                }
            }
        });

        // API 文档折叠/展开
        const apiHeader = document.getElementById('api-header');
        const apiSection = document.querySelector('.api-section');
        if (apiHeader && apiSection) {
            // 默认折叠
            apiSection.classList.add('collapsed');
            apiHeader.addEventListener('click', () => {
                apiSection.classList.toggle('collapsed');
            });
        }

        // 复制按钮
        document.querySelectorAll('.copy-btn').forEach(btn => {
            btn.addEventListener('click', async () => {
                const targetId = btn.dataset.target;
                const target = document.getElementById(targetId);
                if (!target) return;

                try {
                    await navigator.clipboard.writeText(target.textContent);
                    btn.textContent = '已复制';
                    btn.classList.add('copied');
                    setTimeout(() => {
                        btn.textContent = '复制';
                        btn.classList.remove('copied');
                    }, 2000);
                } catch (e) {
                    console.error('复制失败:', e);
                }
            });
        });
    }

    // 启动应用
    document.addEventListener('DOMContentLoaded', init);

})();
