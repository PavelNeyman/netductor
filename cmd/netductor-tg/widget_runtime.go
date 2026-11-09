package main

// Shared TelegramWidget boot — official API (core.telegram.org/bots/widgets):
//   state: TelegramWidget.chatData + event chatDataChanged
//   queries: TelegramWidget.request(data) ≤64 bytes → widget_query
// Do NOT use __tgChatData / WebApp.initDataUnsafe / sendData (Mini App / wrong).

const widgetBootJS = `(function(){
var W=window.TelegramWidget;
function readData(){
  try{
    if(W&&W.chatData!=null&&W.chatData!==''){
      var x=W.chatData;
      return (typeof x==='string')?JSON.parse(x):x;
    }
  }catch(e){}
  return null;
}
function bindPaint(paint){
  function run(){try{paint(readData());}catch(e){}}
  run();
  if(W&&typeof W.onEvent==='function'){
    W.onEvent('chatDataChanged',run);
    W.onEvent('publicDataChanged',run);
  }
}
function sendAct(act){
  act=String(act||'');
  if(act.length>64)act=act.slice(0,64);
  if(W&&typeof W.request==='function'){
    try{W.request(act);}catch(e){}
    return;
  }
  if(W&&typeof W.sendData==='function'){
    try{W.sendData(act);}catch(e){}
  }
}
window.__ndData=readData;
window.__ndBind=bindPaint;
window.__ndSend=sendAct;
})();`

// widgetChatDataMax is Bot API limit for chat_data / public_data (bytes).
const widgetChatDataMax = 1024

func clipWidgetChatData(s string) (string, error) {
	if len(s) <= widgetChatDataMax {
		return s, nil
	}
	return "", fmtWidgetDataTooLarge(len(s))
}

func fmtWidgetDataTooLarge(n int) error {
	return errWidgetDataSize{n}
}

type errWidgetDataSize struct{ n int }

func (e errWidgetDataSize) Error() string {
	return "widget chat_data " + itoa(e.n) + " > 1024"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
