class GroupsListItemView extends KDListItemView

  constructor:(options = {}, data)->
    options.type = "groups"
    super options,data

    {title, slug, body} = @getData()
    @backgroundImage = "http://lorempixel.com/60/60/?#{@utils.getRandomNumber()}"
    @avatar = new KDCustomHTMLView
      tagName : 'img'
      cssClass : 'avatar-image'
      attributes :
        # src : @getData().avatar or "/images/defaultavatar/default.group.128.png"
        src : @getData().avatar or @backgroundImage

    # @settingsButton = new KDButtonViewWithMenu
    #     cssClass    : 'transparent groups-settings-context groups-settings-menu'
    #     title       : ''
    #     icon        : yes
    #     delegate    : @
    #     iconClass   : "arrow"
    #     menu        : @settingsMenu data
    #     callback    : (event)=> @settingsButton.contextMenu event

    # TODO : hide settings button for non-admins
    # @settingsButton.hide()


    @titleLink = new KDCustomHTMLView
      tagName     : 'a'
      attributes  :
        href      : "/#{slug}"
        target    : slug
      pistachio   : '{div{#(title)}}'
      tooltip     :
        title     : title
        direction : 'right'
        placement : 'top'
        selector : 'div.data'
        offset    :
          top     : 6
          left    : -2
        showOnlyWhenOverflowing : yes
      # click       : (event) => @titleReceivedClick event
    , data

    @bodyView = new KDCustomHTMLView
      tagName  : 'div'
      pistachio : '{div{#(body)}}'
      tooltip     :
        title     : body
        direction : 'right'
        placement : 'top'
        selector : 'div.data'
        offset    :
          top     : 6
          left    : -2
        showOnlyWhenOverflowing : yes
    ,data

    @joinButton = new JoinButton
      style           : if data.member then "join-group follow-btn following-topic" else "join-group follow-btn"
      title           : "Join"
      dataPath        : "member"
      defaultState    : if data.member then "Leave" else "Join"
      loader          :
        color         : "#333333"
        diameter      : 18
        top           : 11
      states          : [
        "Join", (callback)->
          data.join (err, response)=>
            @hideLoader()
            unless err
              # @setClass 'following-btn following-topic'
              @setClass 'following-topic'
              @emit 'Joined'
              callback? null
        "Leave", (callback)->
          data.leave (err, response)=>
            @hideLoader()
            unless err
              # @unsetClass 'following-btn following-topic'
              @unsetClass 'following-topic'
              @emit 'Left'
              callback? null
      ]
    , data

    @joinButton.on 'Joined', =>
      @enterButton.show()

    @joinButton.on 'Left', =>
      @enterButton.hide()

    # TODO: hide enter button for non-admins

    # @enterButton = new KDButtonView
    #   cssClass        : 'follow-btn following-btn enter-group hidden'
    #   title           : "Enter"
    #   dataPath        : "member"
    #   # icon : yes
    #   # iconClass : 'enter-group'
    #   callback        : (event)=>
    #     group = @getData().slug
    #     KD.getSingleton('router').handleRoute "#{unless group is 'koding' then '/'+group else ''}/Activity"

    # , data

    @enterLink = new CustomLinkView
      href    : "/#{slug}/Activity"
      target  : slug
      title   : 'Open group'
      click   : @bound 'privateGroupOpenHandler'

  privateGroupOpenHandler: GroupsAppController.privateGroupOpenHandler

  # settingsMenu:(data)->

  #   account        = KD.whoami()
  #   mainController = @getSingleton('mainController')

  #   menu =
  #     'Group settings'  :
  #       callback        : =>
  #         mainController.emit 'EditGroupButtonClicked', @
  #     # 'Permissions'     :
  #     #   callback : =>
  #     #     mainController.emit 'EditPermissionsButtonClicked', @
  #     'My roles'        :
  #       callback        : =>
  #         mainController.emit 'MyRolesRequested', @

  #   # if KD.checkFlag 'super-admin'
  #   #   menu =
  #   #     'MARK USER AS TROLL' :
  #   #       callback : =>
  #   #         mainController.markUserAsTroll data
  #   #     'UNMARK USER AS TROLL' :
  #   #       callback : =>
  #   #         mainController.unmarkUserAsTroll data

  #   return menu

  titleReceivedClick:(event)->
    group = @getData()
    KD.getSingleton('router').handleRoute "/#{group.slug}", state:group
    event.stopPropagation()
    event.preventDefault()
    #KD.getSingleton("appManager").tell "Groups", "createContentDisplay", group

  viewAppended:->
    @setClass "topic-item"

    @setTemplate @pistachio()
    @template.update()

    @$().css "background-image" : @backgroundImage

    # log @titleLink.$()[0]#.innerWidth

    # @$().css backgroundImage : 'url('+(@getData().avatar or "http://lorempixel.com/#{100+@utils.getRandomNumber(300)}/#{50+@utils.getRandomNumber(150)}")+')'


  setFollowerCount:(count)->
    @$('.followers a').html count

  expandItem:->
    return unless @_trimmedBody
    list = @getDelegate()
    $item   = @$()
    $parent = list.$()
    @$clone = $clone = $item.clone()

    pos = $item.position()
    pos.height = $item.outerHeight(no)
    $clone.addClass "clone"
    $clone.css pos
    $clone.css "background-color" : "white"
    $clone.find('.topictext article').html @getData().body
    $parent.append $clone
    $clone.addClass "expand"
    $clone.on "mouseleave",=>@collapseItem()

  collapseItem:->
    return unless @_trimmedBody
    # @$clone.remove()

  pistachio:-> # {article{#(body)}}
    """
    <div class="topictext">
      <span class="avatar">{{>@avatar}}</span>
      <div class="content">
      {h3{> @titleLink}}
      {article{> @bodyView}}
      </div>
      <div class="topicmeta clearfix">
        <div class="topicstats">
          <p class="members">
            <span class="icon"></span>
            <a href="#">{{#(counts.members) or 0}}</a> Members
          </p>
        </div>
      </div>
    </div>
    """

class ModalGroupsListItem extends TopicsListItemView

  constructor:(options,data)->

    super options,data

    @titleLink = new TagLinkView
      expandable: no
      click     : => @getDelegate().emit "CloseTopicsModal"
    , data

  pistachio:->
    """
    <div class="topictext">
      <div class="topicmeta">
        <div class="button-container">{{> @joinButton}}</div>
        {{> @titleLink}}
        <div class="stats">
          <p class="members">
            <span class="icon"></span>{{#(counts.members) or 0}} Members
          </p>
        </div>
      </div>
    </div>
    """

class GroupsListItemViewEditable extends GroupsListItemView

  constructor:(options = {}, data)->

    options.editable = yes
    options.type     = "topics"

    super options, data
