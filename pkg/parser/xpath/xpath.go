package xpath

import (
  "bytes"
  "context"
  "fmt"
  "strings"

  "github.com/antchfx/htmlquery"
  "github.com/go-resty/resty/v2"
  "github.com/ushakovn/outfit/pkg/stringer"
  "golang.org/x/net/html"
)

type ShiftNodePos int

const (
  ShiftNone          ShiftNodePos = 0
  ShiftToFirstChild  ShiftNodePos = 1
  ShiftToLastChild   ShiftNodePos = 2
  ShiftToPrevSibling ShiftNodePos = 3
  ShiftToNextSibling ShiftNodePos = 4
)

type HtmlDocument struct {
  Node *html.Node
  Url  string
}

type Dependencies struct {
  Client *resty.Client
}

type Parser struct {
  deps Dependencies
}

func NewParser(deps Dependencies) *Parser {
  return &Parser{
    deps: deps,
  }
}

func (p *Parser) GetHtmlNode(ctx context.Context, url string) (*html.Node, error) {
  resp, err := p.deps.Client.R().SetContext(ctx).Get(url)
  if err != nil {
    return nil, fmt.Errorf("p.deps.Telegram.R().Get: %w", err)
  }

  node, err := html.Parse(bytes.NewReader(resp.Body()))
  if err != nil {
    return nil, fmt.Errorf("html.Parse(resp.RawBody()): %w", err)
  }

  return node, nil
}

func (p *Parser) GetHtmlDoc(ctx context.Context, url string) (*HtmlDocument, error) {
  htmlDoc, err := p.GetHtmlNode(ctx, url)
  if err != nil {
    return nil, fmt.Errorf("p.GetHtmlNode: %w", err)
  }

  return &HtmlDocument{
    Node: htmlDoc,
    Url:  url,
  }, nil
}

func HandleElement(doc *HtmlDocument, xpath string, handler func(*html.Node)) {
  nodes := htmlquery.Find(doc.Node, xpath)

  for _, node := range nodes {
    if node == nil {
      continue
    }
    handler(node)
  }
}

func ZipElements(f, s []*html.Node, handler func(*html.Node, *html.Node) error) error {
  if len(f) != len(s) {
    return fmt.Errorf("lengths not equal")
  }

  for i := 0; i < len(f); i++ {
    err := handler(f[i], s[i])
    if err != nil {
      return err
    }
  }

  return nil
}

func FindElement(doc *HtmlDocument, xpath string, handler func(node *html.Node) bool) (*html.Node, bool) {
  nodes := CollectElements(doc, xpath)

  for _, node := range nodes {
    if node == nil {
      continue
    }
    if handler(node) {
      return node, true
    }
  }

  return nil, false
}

func CollectElements(doc *HtmlDocument, xpath string) []*html.Node {
  var nodes []*html.Node

  HandleElement(doc, xpath, func(n *html.Node) {
    nodes = append(nodes, n)
  })

  return nodes
}

func GetFirstElement(doc *HtmlDocument, xpath string) *html.Node {
  var firstNode *html.Node
  nodes := htmlquery.Find(doc.Node, xpath)

  for _, node := range nodes {
    if node == nil {
      continue
    }
    firstNode = node

    break
  }

  return firstNode
}

func GetAttributeContains(node *html.Node, attrKey string) (string, bool) {
  if node == nil {
    return "", false
  }

  for _, attr := range node.Attr {
    if !strings.Contains(attr.Key, attrKey) {
      continue
    }
    return stringer.StripTags(attr.Val), true
  }

  return "", false
}

func GetAttribute(node *html.Node, attrKey string) (string, bool) {
  if node == nil {
    return "", false
  }

  for _, attr := range node.Attr {
    if attr.Key != attrKey {
      continue
    }
    return stringer.StripTags(attr.Val), true
  }

  return "", false
}

func GetContent(node *html.Node, shift ShiftNodePos) (string, bool) {
  node = ShiftNode(node, shift)
  if node == nil {
    return "", false
  }

  content := stringer.StripTags(node.Data)
  content = html.UnescapeString(content)

  return content, !stringer.IsEmptyStr(content)
}

func RenderContent(node *html.Node, shift ShiftNodePos) (string, error) {
  node = ShiftNode(node, shift)
  if node == nil {
    return "", nil
  }

  b := &bytes.Buffer{}
  if err := html.Render(b, node); err != nil {
    return "", err
  }

  content := b.String()
  separator := ". "

  content = stringer.ReplaceRepeatSeparators(stringer.StripTags(content), separator)
  content = html.UnescapeString(content)

  return content, nil
}

func ShiftNode(node *html.Node, shift ShiftNodePos) *html.Node {
  if node == nil {
    return node
  }
  switch shift {
  case ShiftToFirstChild:
    return node.FirstChild
  case ShiftToLastChild:
    return node.LastChild
  case ShiftToPrevSibling:
    return node.PrevSibling
  case ShiftToNextSibling:
    return node.NextSibling
  default:
    return node
  }
}
