class Panel extends JView

  constructor: (options = {}, data) ->

    options.cssClass = "panel"

    super options, data

    @headerButtons  = {}
    @panesContainer = []
    @panes          = []
    @panesByName    = {}
    @header         = new KDCustomHTMLView

    {title}         = options
    buttonsLength   = options.buttons?.length

    @createHeader title     if title or buttonsLength
    @createHeaderButtons()  if buttonsLength
    @createHeaderHint()     if options.hint

    @createLayout()

  createHeader: (title = "") ->
    @header     = new KDView
      cssClass  : "inner-header"
      partial   : """<span class="title">#{title}</span>"""

  createHeaderButtons: ->
    # TODO: fatihacet - DRY
    @getOptions().buttons.forEach (buttonOptions) =>
      if buttonOptions.itemClass
        Klass = buttonOptions.itemClass
        delete buttonOptions.itemClass
        buttonOptions.callback = buttonOptions.callback?.bind this, this, @getDelegate()

        buttonView = new Klass buttonOptions
      else
        buttonOptions.callback = buttonOptions.callback?.bind this, this, @getDelegate()
        buttonView = new KDButtonView buttonOptions

      @headerButtons[buttonOptions.title] = buttonView
      @header.addSubView buttonView

  createHeaderHint: ->
    @header.addSubView new KDCustomHTMLView
      cssClass  : "help"
      tooltip   :
        title   : "Need help?"
      click     : => @showHintModal()

  createLayout: ->
    {pane, layout} = @getOptions()
    @container     = new KDView
      cssClass     : "panel-container"

    if pane
      newPane = @createPane pane
      @container.addSubView newPane
      @getDelegate().emit "AllPanesAddedToPanel", @, [newPane]
    else if layout
      @layoutContainer = new WorkspaceLayout
        delegate      : @
        layoutOptions : layout

      @container.addSubView @layoutContainer
    else
      warn "no layout config or pane passed to create a panel"

  createPane: (paneOptions) ->
    PaneClass = @getPaneClass paneOptions
    pane      = new PaneClass paneOptions

    @panesByName[paneOptions.name] = pane  if paneOptions.name

    @panes.push pane
    @emit "NewPaneCreated", pane
    return pane

  getPaneClass: (paneOptions) ->
    paneType             = paneOptions.type
    paneOptions.delegate = @

    if paneType is "custom"
      PaneClass = paneOptions.paneClass
    else
      PaneClass = @findPaneClass paneType

    return unless PaneClass
      new Error "PaneClass is not defined for \"#{paneOptions.type}\" pane type"

    return PaneClass

  findPaneClass: (paneType) ->
    paneTypesToPaneClass =
      "terminal"         : @TerminalPaneClass
      "editor"           : @EditorPaneClass
      "video"            : @VideoPaneClass
      "preview"          : @PreviewPaneClass
      "finder"           : @FinderPaneClass
      "tabbedEditor"     : @TabbedEditorPaneClass
      "paint"            : @PaintPaneClass

    return paneTypesToPaneClass[paneType]

  getPaneByName: (name) ->
    return @panesByName[name] or null

  showHintModal: ->
    options        = @getOptions()
    modal          = new KDModalView
      cssClass     : "workspace-modal"
      overlay      : yes
      title        : options.title
      content      : options.hint
      buttons      :
        Close      :
          title    : "Close"
          cssClass : "modal-cancel"
          callback : -> modal.destroy()

  viewAppended: ->
    super
    @getDelegate().emit "NewPanelAdded", @

  pistachio: ->
    """
      {{> @header}}
      {{> @container}}
    """

Panel::EditorPaneClass        = EditorPane
Panel::TabbedEditorPaneClass  = EditorPane
Panel::TerminalPaneClass      = TerminalPane
Panel::VideoPaneClass         = VideoPane
Panel::PreviewPaneClass       = PreviewPane
Panel::PaintPaneClass         = KDView
