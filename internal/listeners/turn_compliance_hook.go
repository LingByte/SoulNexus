package listeners

// OnTurnPersisted is optional: handlers wire realtime turn censor here.
// Args: callID, tenantID, userText, aiText.
var OnTurnPersisted func(callID string, tenantID uint, userText, aiText string)
