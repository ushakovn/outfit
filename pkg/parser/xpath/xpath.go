package xpath

import (
  "bytes"
  "context"
  "fmt"

  "github.com/antchfx/htmlquery"
  "github.com/go-resty/resty/v2"
  "github.com/ushakovn/outfit/pkg/stringer"
  "golang.org/x/net/html"
)

type NodeShift int

const (
  ShiftNone          NodeShift = 0
  ShiftToFirstChild  NodeShift = 1
  ShiftToLastChild   NodeShift = 2
  ShiftToPrevSibling NodeShift = 3
  ShiftToNextSibling NodeShift = 4
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

func (p *Parser) HandleElement(doc *HtmlDocument, xpath string, handler func(*html.Node)) {
  nodes := htmlquery.Find(doc.Node, xpath)
  for _, node := range nodes {
    if node == nil {
      continue
    }
    handler(node)
  }
}

func (p *Parser) ZipElements(f, s []*html.Node, handler func(*html.Node, *html.Node) error) error {
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

func (p *Parser) FindElement(doc *HtmlDocument, xpath string, handler func(node *html.Node) bool) (*html.Node, bool) {
  nodes := p.CollectElements(doc, xpath)

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

func (p *Parser) CollectElements(doc *HtmlDocument, xpath string) []*html.Node {
  var nodes []*html.Node
  p.HandleElement(doc, xpath, func(n *html.Node) {
    nodes = append(nodes, n)
  })
  return nodes
}

func (p *Parser) GetFirstElement(doc *HtmlDocument, xpath string) *html.Node {
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

func (p *Parser) GetAttribute(node *html.Node, attrKey string) (string, bool) {
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

func (p *Parser) GetContent(node *html.Node, shift NodeShift) (string, bool) {
  node = p.NodeShift(node, shift)
  if node == nil {
    return "", false
  }
  return stringer.StripTags(node.Data), !stringer.IsEmptyStr(node.Data)
}

func (p *Parser) ExtractNodeContent(node *html.Node, shift NodeShift) (string, error) {
  node = p.NodeShift(node, shift)
  if node == nil {
    return "", nil
  }
  b := &bytes.Buffer{}
  if err := html.Render(b, node); err != nil {
    return "", err
  }
  content := b.String()
  separator := ". "

  content = stringer.TrimRepeatSeparators(stringer.StripTags(content), separator)
  content = html.UnescapeString(content)

  return content, nil
}

func (p *Parser) NodeShift(node *html.Node, shift NodeShift) *html.Node {
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
