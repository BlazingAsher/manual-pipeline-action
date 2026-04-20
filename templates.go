package main

import (
	"html/template"
	"time"
)

type TemplateData struct {
	UUID     string
	Question string
	Options  []string
	Answer   string
	Answered bool
}

type QuestionRow struct {
	ID         string
	Question   string
	Options    []string
	Answer     *string
	AnsweredAt *time.Time
	CreatedAt  time.Time
}

type AdminData struct {
	Questions []QuestionRow
}

const interactionHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Pipeline Question</title>
  <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="min-h-screen bg-gray-50 flex items-center justify-center p-4">
  <div class="bg-white rounded-2xl shadow-md p-8 max-w-lg w-full">
    {{if .Answered}}
      <div class="flex items-center gap-3 mb-2">
        <svg class="w-6 h-6 text-green-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
        </svg>
        <p class="text-green-700 font-semibold text-lg">Response recorded</p>
      </div>
      <p class="text-gray-600 mt-1">You selected: <span class="font-medium text-gray-900">{{.Answer}}</span></p>
      <p class="text-gray-400 text-sm mt-4">You can close this tab.</p>
    {{else}}
      <p class="text-xs font-semibold uppercase tracking-widest text-indigo-500 mb-2">Pipeline is waiting for your input</p>
      <h1 class="text-xl font-bold text-gray-900 mb-6">{{.Question}}</h1>
      <form method="POST" action="/i/{{.UUID}}/respond" class="flex flex-col gap-3">
        {{range .Options}}
        <button
          name="answer"
          value="{{.}}"
          class="w-full py-2.5 px-4 bg-indigo-600 text-white font-medium rounded-lg hover:bg-indigo-700 active:bg-indigo-800 transition-colors"
        >{{.}}</button>
        {{end}}
      </form>
    {{end}}
  </div>
</body>
</html>`

const loginHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Admin Login</title>
  <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="min-h-screen bg-gray-50 flex items-center justify-center p-4">
  <div class="bg-white rounded-2xl shadow-md p-8 w-full max-w-sm">
    <h1 class="text-xl font-bold text-gray-900 mb-6">Admin Login</h1>
    {{if .Invalid}}
    <p class="text-red-600 text-sm mb-4">Invalid token.</p>
    {{end}}
    <form method="POST" action="/admin/login" class="flex flex-col gap-4">
      <input
        type="password"
        name="token"
        placeholder="Auth token"
        required
        class="w-full border border-gray-300 rounded-lg px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
      />
      <button
        type="submit"
        class="w-full py-2.5 px-4 bg-indigo-600 text-white font-medium rounded-lg hover:bg-indigo-700 active:bg-indigo-800 transition-colors"
      >Sign in</button>
    </form>
  </div>
</body>
</html>`

const adminHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Admin — Questions</title>
  <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="min-h-screen bg-gray-50 p-8">
  <div class="max-w-6xl mx-auto">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-900">Questions</h1>
      <span class="text-sm text-gray-400">{{len .Questions}} total</span>
    </div>
    {{if not .Questions}}
    <p class="text-gray-400 text-sm">No questions yet.</p>
    {{else}}
    <div class="overflow-x-auto rounded-2xl shadow-sm border border-gray-200">
      <table class="min-w-full divide-y divide-gray-200 text-sm">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-4 py-3 text-left font-semibold text-gray-600">ID</th>
            <th class="px-4 py-3 text-left font-semibold text-gray-600">Question</th>
            <th class="px-4 py-3 text-left font-semibold text-gray-600">Options</th>
            <th class="px-4 py-3 text-left font-semibold text-gray-600">Answer</th>
            <th class="px-4 py-3 text-left font-semibold text-gray-600">Answered At</th>
            <th class="px-4 py-3 text-left font-semibold text-gray-600">Created At</th>
            <th class="px-4 py-3 text-left font-semibold text-gray-600">Link</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-100">
          {{range .Questions}}
          <tr class="hover:bg-gray-50">
            <td class="px-4 py-3 font-mono text-xs text-gray-400">{{.ID}}</td>
            <td class="px-4 py-3 text-gray-900 max-w-xs">{{.Question}}</td>
            <td class="px-4 py-3 text-gray-600">
              {{range .Options}}<span class="inline-block bg-gray-100 rounded px-1.5 py-0.5 text-xs mr-1 mb-1">{{.}}</span>{{end}}
            </td>
            <td class="px-4 py-3">
              {{if .Answer}}
              <span class="inline-flex items-center gap-1 bg-green-50 text-green-700 font-medium rounded px-2 py-0.5 text-xs">
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
                {{.Answer}}
              </span>
              {{else}}
              <span class="inline-block bg-yellow-50 text-yellow-700 rounded px-2 py-0.5 text-xs">pending</span>
              {{end}}
            </td>
            <td class="px-4 py-3 text-gray-500 whitespace-nowrap">
              {{if .AnsweredAt}}{{.AnsweredAt.UTC.Format "2006-01-02 15:04:05"}}{{else}}—{{end}}
            </td>
            <td class="px-4 py-3 text-gray-500 whitespace-nowrap">{{.CreatedAt.UTC.Format "2006-01-02 15:04:05"}}</td>
            <td class="px-4 py-3">
              <a href="/i/{{.ID}}" class="text-indigo-600 hover:underline text-xs">open</a>
            </td>
          </tr>
          {{end}}
        </tbody>
      </table>
    </div>
    {{end}}
  </div>
</body>
</html>`

func parseTemplates() *template.Template {
	t := template.Must(template.New("interaction").Parse(interactionHTML))
	template.Must(t.New("login").Parse(loginHTML))
	template.Must(t.New("admin").Parse(adminHTML))
	return t
}
