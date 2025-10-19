// Get TTS locale for a language object or string
export function languageToLocale(
  language?: string | { tts_locale?: string; name?: string }
): string | undefined {
  if (!language) return undefined;

  // If it's a language object with tts_locale, use that
  if (typeof language === 'object' && language.tts_locale) {
    return language.tts_locale;
  }

  // If it's a language object with name, get the name
  const languageName = typeof language === 'object' ? language.name : language;

  // Otherwise, it's a language name string, return the corresponding locale
  switch ((languageName || '').toLowerCase()) {
    case 'italian':
      return 'it-IT';
    case 'french':
      return 'fr-FR';
    case 'german':
      return 'de-DE';
    case 'russian':
      return 'ru-RU';
    case 'japanese':
      return 'ja-JP';
    case 'chinese':
      return 'zh-CN';
    default:
      return undefined;
  }
}

// Get default TTS voice for a language object or string
export function defaultVoiceForLanguage(
  language?: string | { tts_voice?: string; name?: string }
): string | undefined {
  if (!language) return undefined;

  // If it's a language object with tts_voice, use that
  if (typeof language === 'object' && language.tts_voice) {
    return language.tts_voice;
  }

  // If it's a language object with name, get the name
  const languageName = typeof language === 'object' ? language.name : language;

  // Otherwise, it's a language name string, return the corresponding voice
  switch ((languageName || '').toLowerCase()) {
    case 'italian':
      return 'it-IT-IsabellaNeural';
    case 'french':
      return 'fr-FR-DeniseNeural';
    case 'german':
      return 'de-DE-KatjaNeural';
    case 'russian':
      return 'ru-RU-DariyaNeural';
    case 'japanese':
      return 'ja-JP-NanamiNeural';
    case 'chinese':
      return 'zh-CN-XiaoxiaoNeural';
    default:
      return undefined;
  }
}

export interface EdgeTTSVoiceInfo {
  name?: string;
  short_name?: string;
  display_name?: string;
  Locale?: string;
  locale?: string;
  language?: string;
  Gender?: string;
  gender?: string;
}

export function extractVoiceName(v: EdgeTTSVoiceInfo): string | undefined {
  return v.short_name || v.name || v.display_name;
}
// Sample paragraphs (2-3 sentences) for each supported language so users can
// better evaluate voice quality and prosody.
export function sampleTextForLanguage(
  language?: string | { name?: string }
): string | undefined {
  if (!language) return undefined;

  // If it's a language object, get the name
  const langName = typeof language === 'object' ? language.name : language;

  switch ((langName || '').toLowerCase()) {
    case 'italian':
      return (
        'Ciao! Questo è un esempio di voce. ' +
        'Puoi ascoltare la pronuncia e l' +
        'intonazione in queste brevi frasi. Buon ascolto!'
      );
    case 'french':
      return (
        'Bonjour! Ceci est un exemple de voix. ' +
        'Écoutez la prononciation et le rythme dans ces courtes ' +
        'phrases. Bonne écoute!'
      );
    case 'german':
      return (
        'Hallo! Dies ist ein Sprachbeispiel. ' +
        'Hören Sie auf die Aussprache und den Rhythmus in diesen ' +
        'Sätzen. Viel Spaß beim Anhören!'
      );
    case 'russian':
      return (
        'Привет! Это пример голоса. ' +
        'Обратите внимание на произношение и интонацию в этих ' +
        'коротких предложениях. Приятного прослушивания!'
      );
    case 'japanese':
      return (
        'こんにちは！これは音声のサンプルです。' +
        '短い文で発音と抑揚を確認できます。' +
        'どうぞお楽しみください。'
      );
    case 'chinese':
      return (
        '你好！这是语音示例。' +
        '通过这些简短的句子，您可以听到发音和语调。' +
        '祝您聆听愉快！'
      );
    default:
      return undefined;
  }
}
