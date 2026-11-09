using System;
using System.CodeDom.Compiler;
using System.Collections;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Text.RegularExpressions;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Controls.Primitives;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Markup;
using System.Windows.Threading;
using System.Xml.Linq;
using SpecDesigner.Controls.Controls;
using SpecDesigner.FormDataEditor;
using SpecDesigner.SpecEditor.Helpers;
using SpecDesigner.SpecEditor.Views;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedoCommands;
using SpecDesignerCommon.ViewModel;
using SpecDesignerPreference;

namespace SpecDesigner.SpecEditor
{
	// Token: 0x02000007 RID: 7
	public partial class SpecPropertyEditor : UserControl, INotifyPropertyChanged
	{
		// Token: 0x17000004 RID: 4
		// (get) Token: 0x0600001A RID: 26 RVA: 0x00002381 File Offset: 0x00000581
		// (set) Token: 0x0600001B RID: 27 RVA: 0x00002389 File Offset: 0x00000589
		public FormSpecModel CurrentSpecNode
		{
			get
			{
				return this._currentSpecNode;
			}
			private set
			{
				this._currentSpecNode = value;
				this.OnPropertyChanged("CurrentSpecNode");
			}
		}

		// Token: 0x17000005 RID: 5
		// (get) Token: 0x0600001C RID: 28 RVA: 0x0000239D File Offset: 0x0000059D
		public static SpecPropertyEditor This
		{
			get
			{
				return SpecPropertyEditor._this;
			}
		}

		// Token: 0x0600001D RID: 29 RVA: 0x000023A4 File Offset: 0x000005A4
		public SpecPropertyEditor()
		{
			this.InitializeComponent();
			if (DesignerProperties.GetIsInDesignMode(this))
			{
				return;
			}
			this.Init();
			base.CommandBindings.Add(new CommandBinding(SpecPropertyCommands.ShowZoomsWindowCommand, new ExecutedRoutedEventHandler(SpecPropertyCommands.ExecutedShowZoomsWindow), new CanExecuteRoutedEventHandler(SpecPropertyCommands.CanShowZoomsWindow)));
			base.CommandBindings.Add(new CommandBinding(SpecPropertyCommands.ShowItemsWindowCommand, new ExecutedRoutedEventHandler(this.ExecutedShowItemsWindow), new CanExecuteRoutedEventHandler(this.CanShowItemsWindow)));
			base.CommandBindings.Add(new CommandBinding(SpecPropertyCommands.ShowLocalStringWindowCommand, new ExecutedRoutedEventHandler(this.ExecutedShowLocalStringWindow), new CanExecuteRoutedEventHandler(this.CanShowLocalStringWindow)));
			base.CommandBindings.Add(new CommandBinding(SpecPropertyCommands.ClearContentCommand, new ExecutedRoutedEventHandler(SpecPropertyCommands.ExecutedClearContent), new CanExecuteRoutedEventHandler(SpecPropertyCommands.CanClearContent)));
			this.ComponentPropertyVisibilityMap_Init();
		}

		// Token: 0x0600001E RID: 30 RVA: 0x0000249C File Offset: 0x0000069C
		private void ComponentPropertyVisibilityMap_Init()
		{
			Dictionary<string, string> dictionary = new Dictionary<string, string>();
			Dictionary<string, string> dictionary2 = new Dictionary<string, string>();
			Dictionary<string, string> dictionary3 = new Dictionary<string, string>();
			Dictionary<string, string> dictionary4 = new Dictionary<string, string>();
			Dictionary<string, string> dictionary5 = new Dictionary<string, string>();
			Dictionary<string, string> dictionary6 = new Dictionary<string, string>();
			Dictionary<string, string> dictionary7 = new Dictionary<string, string>();
			Dictionary<string, string> dictionary8 = new Dictionary<string, string>();
			Dictionary<string, string> dictionary9 = new Dictionary<string, string>();
			dictionary2.Add("specProperty_type", "L");
			dictionary3.Add("specProperty_type", "L");
			dictionary4.Add("specProperty_type", "L");
			dictionary5.Add("specProperty_type", "L");
			dictionary6.Add("specProperty_type", "L");
			dictionary7.Add("specProperty_type", "L");
			dictionary2.Add("specProperty_i_zoom", "L");
			dictionary4.Add("specProperty_i_zoom", "L");
			dictionary5.Add("specProperty_i_zoom", "L");
			dictionary6.Add("specProperty_i_zoom", "L");
			dictionary7.Add("specProperty_i_zoom", "L");
			dictionary2.Add("specProperty_c_zoom", "L");
			dictionary4.Add("specProperty_c_zoom", "L");
			dictionary5.Add("specProperty_c_zoom", "L");
			dictionary6.Add("specProperty_c_zoom", "L");
			dictionary7.Add("specProperty_c_zoom", "L");
			dictionary4.Add("specProperty_chk_ref", "L");
			dictionary5.Add("specProperty_chk_ref", "L");
			dictionary6.Add("specProperty_chk_ref", "L");
			dictionary7.Add("specProperty_chk_ref", "L");
			dictionary2.Add("specProperty_items", "L");
			dictionary3.Add("specProperty_items", "L");
			dictionary4.Add("specProperty_items", "L");
			dictionary6.Add("specProperty_items", "L");
			dictionary7.Add("specProperty_items", "L");
			dictionary7.Add("specProperty_default", "L");
			dictionary3.Add("specProperty_max", "L");
			dictionary4.Add("specProperty_max", "L");
			dictionary5.Add("specProperty_max", "L");
			dictionary6.Add("specProperty_max", "L");
			dictionary7.Add("specProperty_max", "L");
			dictionary3.Add("specProperty_min", "L");
			dictionary4.Add("specProperty_min", "L");
			dictionary5.Add("specProperty_min", "L");
			dictionary6.Add("specProperty_min", "L");
			dictionary7.Add("specProperty_min", "L");
			dictionary2.Add("specProperty_ver", "L");
			dictionary3.Add("specProperty_ver", "L");
			dictionary4.Add("specProperty_ver", "L");
			dictionary5.Add("specProperty_ver", "L");
			dictionary6.Add("specProperty_ver", "L");
			dictionary7.Add("specProperty_ver", "L");
			dictionary8.Add("specProperty_formonly_text", "L");
			dictionary.Add("Widget", "L");
			dictionary2.Add("Widget", "L");
			dictionary3.Add("Widget", "L");
			dictionary4.Add("Widget", "L");
			dictionary5.Add("Widget", "L");
			dictionary6.Add("Widget", "L");
			dictionary7.Add("Widget", "L");
			dictionary8.Add("Widget", "L");
			dictionary9.Add("Widget", "L");
			dictionary.Add("comment", "L");
			dictionary2.Add("comment", "L");
			dictionary3.Add("comment", "L");
			dictionary4.Add("comment", "L");
			dictionary5.Add("comment", "L");
			dictionary6.Add("comment", "L");
			dictionary7.Add("comment", "L");
			dictionary8.Add("comment", "L");
			dictionary9.Add("comment", "L");
			dictionary5.Add("items", "L");
			dictionary.Add("case", "L");
			dictionary2.Add("case", "L");
			dictionary3.Add("case", "L");
			dictionary5.Add("case", "L");
			dictionary7.Add("case", "L");
			dictionary.Add("noEntry", "L");
			dictionary.Add("notNull", "L");
			dictionary2.Add("notNull", "L");
			dictionary3.Add("notNull", "L");
			dictionary5.Add("notNull", "L");
			dictionary6.Add("notNull", "L");
			dictionary7.Add("notNull", "L");
			dictionary3.Add("action", "L");
			dictionary3.Add("image", "L");
			dictionary9.Add("image", "L");
			dictionary.Add("scroll", "L");
			dictionary2.Add("scroll", "L");
			dictionary3.Add("scroll", "L");
			dictionary5.Add("scroll", "L");
			dictionary.Add("justify", "L");
			dictionary2.Add("justify", "L");
			dictionary3.Add("justify", "L");
			dictionary4.Add("justify", "L");
			dictionary5.Add("justify", "L");
			dictionary6.Add("justify", "L");
			dictionary7.Add("justify", "L");
			dictionary8.Add("justify", "L");
			dictionary.Add("style", "L");
			dictionary2.Add("style", "L");
			dictionary3.Add("style", "L");
			dictionary4.Add("style", "L");
			dictionary5.Add("style", "L");
			dictionary6.Add("style", "L");
			dictionary7.Add("style", "L");
			dictionary8.Add("style", "L");
			dictionary9.Add("style", "L");
			dictionary.Add("tabIndex", "L");
			dictionary2.Add("tabIndex", "L");
			dictionary3.Add("tabIndex", "L");
			dictionary4.Add("tabIndex", "L");
			dictionary5.Add("tabIndex", "L");
			dictionary6.Add("tabIndex", "L");
			dictionary7.Add("tabIndex", "L");
			dictionary9.Add("tabIndex", "L");
			dictionary.Add("picture", "L");
			dictionary2.Add("picture", "L");
			dictionary3.Add("picture", "L");
			dictionary6.Add("picture", "L");
			dictionary.Add("format", "L");
			dictionary6.Add("format", "L");
			dictionary.Add("aggregate", "L");
			dictionary2.Add("aggregate", "L");
			dictionary3.Add("aggregate", "L");
			dictionary4.Add("aggregate", "L");
			dictionary5.Add("aggregate", "L");
			dictionary6.Add("aggregate", "L");
			dictionary7.Add("aggregate", "L");
			dictionary8.Add("aggregate", "L");
			dictionary.Add("imageColumn", "L");
			dictionary2.Add("imageColumn", "L");
			dictionary3.Add("imageColumn", "L");
			dictionary.Add("sizePolicy", "L");
			dictionary2.Add("sizePolicy", "L");
			dictionary3.Add("sizePolicy", "L");
			dictionary4.Add("sizePolicy", "L");
			dictionary5.Add("sizePolicy", "L");
			dictionary6.Add("sizePolicy", "L");
			dictionary7.Add("sizePolicy", "L");
			dictionary8.Add("sizePolicy", "L");
			dictionary9.Add("sizePolicy", "L");
			dictionary7.Add("stretch", "L");
			dictionary.Add("gridWidth", "L");
			dictionary2.Add("gridWidth", "L");
			dictionary3.Add("gridWidth", "L");
			dictionary4.Add("gridWidth", "L");
			dictionary5.Add("gridWidth", "L");
			dictionary6.Add("gridWidth", "L");
			dictionary7.Add("gridWidth", "L");
			dictionary8.Add("gridWidth", "L");
			dictionary9.Add("gridWidth", "L");
			dictionary.Add("gridHeight", "L");
			dictionary2.Add("gridHeight", "L");
			dictionary3.Add("gridHeight", "L");
			dictionary4.Add("gridHeight", "L");
			dictionary5.Add("gridHeight", "L");
			dictionary6.Add("gridHeight", "L");
			dictionary7.Add("gridHeight", "L");
			dictionary8.Add("gridHeight", "L");
			dictionary9.Add("gridHeight", "L");
			dictionary.Add("posX", "L");
			dictionary2.Add("posX", "L");
			dictionary3.Add("posX", "L");
			dictionary4.Add("posX", "L");
			dictionary5.Add("posX", "L");
			dictionary6.Add("posX", "L");
			dictionary7.Add("posX", "L");
			dictionary8.Add("posX", "L");
			dictionary9.Add("posX", "L");
			dictionary.Add("posY", "L");
			dictionary2.Add("posY", "L");
			dictionary3.Add("posY", "L");
			dictionary4.Add("posY", "L");
			dictionary5.Add("posY", "L");
			dictionary6.Add("posY", "L");
			dictionary7.Add("posY", "L");
			dictionary8.Add("posY", "L");
			dictionary9.Add("posY", "L");
			dictionary.Add("color", "L");
			dictionary7.Add("scrollBars", "L");
			dictionary.Add("tag", "L");
			dictionary2.Add("tag", "L");
			dictionary3.Add("tag", "L");
			dictionary4.Add("tag", "L");
			dictionary5.Add("tag", "L");
			dictionary6.Add("tag", "L");
			dictionary7.Add("tag", "L");
			dictionary8.Add("tag", "L");
			dictionary9.Add("tag", "L");
			dictionary7.Add("wantNoReturns", "L");
			SpecPropertyEditor.ComponentPropertyVisibilityMap.Add("Reference", dictionary);
			SpecPropertyEditor.ComponentPropertyVisibilityMap.Add("Edit", dictionary2);
			SpecPropertyEditor.ComponentPropertyVisibilityMap.Add("ButtonEdit", dictionary3);
			SpecPropertyEditor.ComponentPropertyVisibilityMap.Add("CheckBox", dictionary4);
			SpecPropertyEditor.ComponentPropertyVisibilityMap.Add("ComboBox", dictionary5);
			SpecPropertyEditor.ComponentPropertyVisibilityMap.Add("DateEdit", dictionary6);
			SpecPropertyEditor.ComponentPropertyVisibilityMap.Add("TextEdit", dictionary7);
			SpecPropertyEditor.ComponentPropertyVisibilityMap.Add("Label", dictionary8);
			SpecPropertyEditor.ComponentPropertyVisibilityMap.Add("ProgRelField", dictionary9);
		}

		// Token: 0x0600001F RID: 31 RVA: 0x00003128 File Offset: 0x00001328
		public SpecificationInfo GetSpecInfo()
		{
			if (this.CurrentSpecNode == null)
			{
				return null;
			}
			if (SettingManager.Get().GetTzpManger(this.CurrentSpecNode.Key) != null)
			{
				return SettingManager.Get().GetTzpManger(this.CurrentSpecNode.Key).SpecificationInfo;
			}
			return null;
		}

		// Token: 0x06000020 RID: 32 RVA: 0x00003167 File Offset: 0x00001367
		private void SpecPropertyEditor_Loaded(object sender, RoutedEventArgs e)
		{
			base.Loaded -= this.SpecPropertyEditor_Loaded;
			this.RenderPropertyGrid(null);
		}

		// Token: 0x06000021 RID: 33 RVA: 0x00003184 File Offset: 0x00001384
		public void ProgramSelectionChanged(ProgramSelectionChangedEventArgs args)
		{
			if (args.NewProgramKey == null)
			{
				this.Init();
				return;
			}
			this._programKey = args.NewProgramKey;
			if (this.LastSpecArgs.ContainsKey(args.NewProgramKey.Program))
			{
				IEnumerable<string> enumerable = this.LastSpecArgs[args.NewProgramKey.Program];
				if (enumerable.Count<string>() == 1)
				{
					SpecArgs specArgs = new SpecArgs(enumerable.FirstOrDefault<string>(), ComponentType.Unknown, args.NewProgramKey);
					this.LoadSpecification(specArgs);
					return;
				}
				if (enumerable.Count<string>() > 1)
				{
					MultiSelectionArgs multiSelectionArgs = new MultiSelectionArgs(args.NewProgramKey, enumerable);
					this.LoadSpecificationWithComponents(multiSelectionArgs);
					return;
				}
			}
			else
			{
				this.Init();
			}
		}

		// Token: 0x06000022 RID: 34 RVA: 0x0000322C File Offset: 0x0000142C
		public void SubscribeTzpFileClosed(PackageKey programKey)
		{
			if (this.CurrentSpecNode != null && this.CurrentSpecNode.Key == programKey)
			{
				this.Init();
			}
			if (this.LastSpecArgs.ContainsKey(programKey.Program))
			{
				this.LastSpecArgs.Remove(programKey.Program);
			}
		}

		// Token: 0x06000023 RID: 35 RVA: 0x0000327F File Offset: 0x0000147F
		public void RenderPropertyGrid(object nullObj)
		{
		}

		// Token: 0x06000024 RID: 36 RVA: 0x00003284 File Offset: 0x00001484
		public void Init()
		{
			this.CurrentSpecNode = null;
			this.FieldExcludeSetting.Visibility = Visibility.Collapsed;
			this.FieldExcludeSetting.DataContext = Binding.DoNothing;
			this.FieldSpecGroup1.Visibility = Visibility.Collapsed;
			this.FieldSpecGroup1.DataContext = Binding.DoNothing;
			this.ActionSpecGroup.Visibility = Visibility.Collapsed;
			this.ActionSpecGroup.DataContext = Binding.DoNothing;
			this.FormonlySpecGroup.Visibility = Visibility.Collapsed;
			this.FormonlySpecGroup.DataContext = Binding.DoNothing;
			this.GeneroComponentAttrGroup.Visibility = Visibility.Collapsed;
			this.GeneroComponentAttrGroup.DataContext = Binding.DoNothing;
			this.ToolBarActionSpecGroup.Visibility = Visibility.Collapsed;
			this.ToolBarActionSpecGroup.DataContext = Binding.DoNothing;
			this.MultiLangSpecGroup.Visibility = Visibility.Collapsed;
			this.MultiLangSpecGroup.DataContext = Binding.DoNothing;
			this.RefFieldSpecGroup.Visibility = Visibility.Collapsed;
			this.RefFieldSpecGroup.DataContext = Binding.DoNothing;
			this.TreeSpecGroup.Visibility = Visibility.Collapsed;
			this.TreeSpecGroup.DataContext = Binding.DoNothing;
			this.ProgRelFieldGroup.Visibility = Visibility.Collapsed;
			this.ProgRelFieldGroup.DataContext = Binding.DoNothing;
			this.HelpCodeSpecGroup.Visibility = Visibility.Collapsed;
			this.HelpCodeSpecGroup.DataContext = Binding.DoNothing;
			this.citeStdGrid.Visibility = Visibility.Collapsed;
			this.CommonGeneroComponentAttrGroup.Visibility = Visibility.Collapsed;
			this.CommonGeneroComponentAttrGroup.DataContext = Binding.DoNothing;
		}

		// Token: 0x06000025 RID: 37 RVA: 0x0000356C File Offset: 0x0000176C
		public void LoadSpecificationWithComponents(MultiSelectionArgs args)
		{
			SpecificationInfo info = SettingManager.Get().GetTzpManger(args.ProgramKey).SpecificationInfo;
			if (info == null)
			{
				return;
			}
			base.Dispatcher.BeginInvoke(new Action(delegate
			{
				this.Init();
				if (args.Components.Count<string>() > 1)
				{
					List<XmlElement> list = new List<XmlElement>();
					foreach (string text in args.Components)
					{
						FormSpecModel formSpecModel = info.FindNodeByName(text);
						if (formSpecModel != null)
						{
							list.Add(formSpecModel.GeneroComponent);
						}
					}
					this.CommonGeneroComponentAttrGroup.Visibility = Visibility.Visible;
					this.CommonGeneroComponentAttrGroup.DataContext = list;
					lock (((ICollection)this.LastSpecArgs).SyncRoot)
					{
						if (this.LastSpecArgs.ContainsKey(args.ProgramKey.Program))
						{
							this.LastSpecArgs[args.ProgramKey.Program] = new List<string>(list.Select<XmlElement, string>((XmlElement e) => e.Name));
						}
					}
					EventAggregatorManager.Global.GetEvent<SpecNodeLoadedEvent>().Publish(null);
				}
			}), DispatcherPriority.Background, new object[0]);
		}

		// Token: 0x06000026 RID: 38 RVA: 0x00003958 File Offset: 0x00001B58
		public void LoadSpecification(SpecArgs args)
		{
			if (args == null)
			{
				base.Dispatcher.BeginInvoke(new Action(delegate
				{
					this.Init();
					EventAggregatorManager.Global.GetEvent<SpecNodeLoadedEvent>().Publish(null);
				}), new object[0]);
				return;
			}
			SpecificationInfo info = SettingManager.Get().GetTzpManger(args.ProgramKey).SpecificationInfo;
			if (info == null)
			{
				return;
			}
			base.Dispatcher.BeginInvoke(new Action(delegate
			{
				this.Init();
				FormSpecModel formSpecModel = info.FindNodeByName(args.Name);
				if (formSpecModel == null)
				{
					IEnumerable<SpecActionNode> enumerable = info.Actions.Where<SpecActionNode>((SpecActionNode a) => a.Name == args.Name);
					SpecActionNode specActionNode;
					if (enumerable.Count<SpecActionNode>() == 1)
					{
						specActionNode = enumerable.FirstOrDefault<SpecActionNode>();
					}
					else
					{
						specActionNode = enumerable.Where<SpecActionNode>((SpecActionNode a) => (a.Status & SpecStatus.DELETE) != SpecStatus.DELETE).FirstOrDefault<SpecActionNode>();
					}
					if (specActionNode != null)
					{
						this.showActionSpecGroupDataContext(specActionNode);
						this.showIsCitedBindingDataContext(specActionNode);
						EventAggregatorManager.Global.GetEvent<SpecNodeLoadedEvent>().Publish(new SpecNodeLoadedEventArgs(specActionNode, specActionNode.ProgramKey));
					}
					return;
				}
				this.CurrentSpecNode = formSpecModel;
				this.showGeneroComponentAttrDataContext();
				this.showIsCitedBindingDataContext(formSpecModel.SpecNode);
				this.showExcludeSettingAndBindingDataContext();
				this.showTreeUIAndBindingDataContext();
				this.showMultiLangFieldUIAndBindingDataContext();
				this.showActionUIAndBindingDataContext();
				this.showFieldUIAndBindingDataContext();
				this.showProgRelFieldGroup();
				this.showReferenceFieldUIandBidningDataContext();
				this.showHelpCodeBidningDataContext();
				AbstractSpecNode abstractSpecNode = null;
				switch (this.CurrentSpecNode.SpecNodeType)
				{
				case SpecNodeType.ACTION:
				case SpecNodeType.TOOLBAR:
					abstractSpecNode = this.CurrentSpecNode.SpecAction;
					goto IL_023E;
				case SpecNodeType.FORMONLY:
					abstractSpecNode = this.CurrentSpecNode.SpecField;
					goto IL_023E;
				case SpecNodeType.PROGREL:
					abstractSpecNode = this.CurrentSpecNode.SpecProgRel;
					goto IL_023E;
				case SpecNodeType.REFERENCE:
					abstractSpecNode = this.CurrentSpecNode.SpecReference;
					goto IL_023E;
				case SpecNodeType.MULTILANG:
					abstractSpecNode = this.CurrentSpecNode.SpecMultiLang;
					goto IL_023E;
				case SpecNodeType.TREE:
					if (this.CurrentSpecNode.Name == "s_browse")
					{
						abstractSpecNode = this.CurrentSpecNode.SpecTree;
						goto IL_023E;
					}
					abstractSpecNode = this.CurrentSpecNode.SpecField;
					goto IL_023E;
				}
				abstractSpecNode = this.CurrentSpecNode.SpecField;
				IL_023E:
				lock (((ICollection)this.LastSpecArgs).SyncRoot)
				{
					if (this.LastSpecArgs.ContainsKey(args.ProgramKey.Program))
					{
						this.LastSpecArgs[args.ProgramKey.Program] = new string[] { formSpecModel.Name };
					}
					else
					{
						this.LastSpecArgs.Add(args.ProgramKey.Program, new string[] { formSpecModel.Name });
					}
				}
				EventAggregatorManager.Global.GetEvent<SpecNodeLoadedEvent>().Publish(new SpecNodeLoadedEventArgs(abstractSpecNode, this.CurrentSpecNode.Key));
			}), DispatcherPriority.Background, new object[0]);
		}

		// Token: 0x06000027 RID: 39 RVA: 0x000039F0 File Offset: 0x00001BF0
		private void citeStdCheckBox_PreviewMouseLeftButtonDown(object sender, MouseButtonEventArgs e)
		{
			bool flag = this.citeStdCheckBox.IsChecked != null && this.citeStdCheckBox.IsChecked.Value;
			e.Handled = true;
			AbstractSpecNode abstractSpecNode = this.citeStdCheckBox.DataContext as AbstractSpecNode;
			string text = ((!flag) ? (Application.Current.FindResource("specEditor_confirmCopyTsd") as string) : (Application.Current.FindResource("specEditor_confirmCancelCopyTsd") as string));
			text = string.Format(text, abstractSpecNode.Name);
			MessageBoxResult messageBoxResult = DesignerMessageBox.Show(text, Application.Current.FindResource("Message_Inform") as string, MessageBoxButton.YesNo);
			if (messageBoxResult == MessageBoxResult.Yes)
			{
				SpecCitedUndoRedoCommand specCitedUndoRedoCommand = new SpecCitedUndoRedoCommand(abstractSpecNode, !flag);
				SettingManager.Get().GetUndoRedoManager(abstractSpecNode.ProgramKey).AddThenExecute(specCitedUndoRedoCommand);
			}
		}

		// Token: 0x06000028 RID: 40 RVA: 0x00003AC4 File Offset: 0x00001CC4
		private void showIsCitedBindingDataContext(AbstractSpecNode node)
		{
			if (node == null)
			{
				return;
			}
			if (SettingManager.Get().GetTzpManger(node.ProgramKey).IsStandardProgram)
			{
				return;
			}
			FormSpecModel formSpecModel = SettingManager.Get().GetTzpManger(node.ProgramKey).SpecificationInfo.FindNodeByName(node.Name);
			if (formSpecModel == null)
			{
				this.citeStdGrid.DataContext = node;
				this.citeStdGrid.Visibility = Visibility.Visible;
				return;
			}
			if (formSpecModel.GeneroComponent.IsCantDel)
			{
				return;
			}
			switch (formSpecModel.SpecNodeType)
			{
			case SpecNodeType.FORMONLY:
			case SpecNodeType.PROGREL:
				return;
			default:
				this.citeStdGrid.DataContext = node;
				this.citeStdGrid.Visibility = Visibility.Visible;
				return;
			}
		}

		// Token: 0x06000029 RID: 41 RVA: 0x00003B6C File Offset: 0x00001D6C
		private void showExcludeSettingAndBindingDataContext()
		{
			if (this.CurrentSpecNode == null)
			{
				return;
			}
			if (this.CurrentSpecNode.GeneroComponent.IsCantDel)
			{
				return;
			}
			ComponentType componentType = ((this.CurrentSpecNode.GeneroComponent.Parent == null) ? ComponentType.Unknown : this.CurrentSpecNode.GeneroComponent.Parent.Type);
			ComponentType componentType2 = componentType;
			if (componentType2 != ComponentType.Unknown)
			{
				switch (componentType2)
				{
				case ComponentType.ScrollGrid:
				case ComponentType.Table:
				case ComponentType.Tree:
					break;
				default:
					if (SpecificationInfo.AllowSaveWidgets.Contains(this.CurrentSpecNode.Type.ToString()) || this.CurrentSpecNode.Type == ComponentType.Table || this.CurrentSpecNode.Type == ComponentType.Tree || this.CurrentSpecNode.Type == ComponentType.ScrollGrid)
					{
						this.FieldExcludeSetting.DataContext = this.CurrentSpecNode;
						this.FieldExcludeSetting.SetBinding(UIElement.VisibilityProperty, new Binding("StandardView")
						{
							Source = PreferenceManager.Current.Settings,
							Converter = new BooleanToVisibilityConverter()
						});
					}
					return;
				}
			}
		}

		// Token: 0x0600002A RID: 42 RVA: 0x00003C73 File Offset: 0x00001E73
		private void showTreeUIAndBindingDataContext()
		{
			if (this.CurrentSpecNode == null)
			{
				return;
			}
			if (this.CurrentSpecNode.SpecNodeType != SpecNodeType.TREE)
			{
				return;
			}
			this.TreeSpecGroup.DataContext = this.CurrentSpecNode.SpecTree;
			this.TreeSpecGroup.Visibility = Visibility.Visible;
		}

		// Token: 0x0600002B RID: 43 RVA: 0x00003CB0 File Offset: 0x00001EB0
		private void showMultiLangFieldUIAndBindingDataContext()
		{
			if (this.CurrentSpecNode == null)
			{
				return;
			}
			if (this.CurrentSpecNode.SpecMultiLang == null)
			{
				return;
			}
			this.MultiLangSpecGroup.DataContext = this.CurrentSpecNode.SpecMultiLang;
			this.langDFComboBox.ItemsSource = this.GetSpecInfo().GetDependFieldsSource(this.CurrentSpecNode.GeneroComponent).ToList<string>();
			this.MultiLangSpecGroup.Visibility = Visibility.Visible;
		}

		// Token: 0x0600002C RID: 44 RVA: 0x00003D1C File Offset: 0x00001F1C
		private void showGeneroComponentAttrDataContext()
		{
			if (this.CurrentSpecNode == null)
			{
				return;
			}
			if (this.CurrentSpecNode.GeneroComponent.NodeName == ComponentType.Phantom.ToString())
			{
				return;
			}
			this.GeneroComponentAttrGroup.DataContext = this.CurrentSpecNode.GeneroComponent;
			this.GeneroComponentAttrGroup.Visibility = Visibility.Visible;
			this.noEntryCheckBox.IsEnabled = this.CurrentSpecNode.SpecNodeType != SpecNodeType.PROGREL;
		}

		// Token: 0x0600002D RID: 45 RVA: 0x00003D98 File Offset: 0x00001F98
		private void showFieldUIAndBindingDataContext()
		{
			if (this.CurrentSpecNode == null)
			{
				return;
			}
			SpecNodeType specNodeType = this.CurrentSpecNode.SpecNodeType;
			if (specNodeType != SpecNodeType.FIELD)
			{
				switch (specNodeType)
				{
				case SpecNodeType.FORMONLY:
				case SpecNodeType.NONE:
					this.FormonlySpecGroup.Visibility = Visibility.Visible;
					this.FormonlySpecGroup.DataContext = this.CurrentSpecNode.GeneroComponent;
					this.showTableAssociationComboBox();
					return;
				case SpecNodeType.PROGREL:
					break;
				default:
					return;
				}
			}
			else
			{
				this.FieldSpecGroup1.DataContext = this.CurrentSpecNode.SpecField;
				this.FieldSpecGroup1.Visibility = Visibility.Visible;
				string codeTemplate = this.GetSpecInfo().GetCodeTemplate();
				if (codeTemplate.Equals("P", StringComparison.CurrentCultureIgnoreCase) || codeTemplate.Equals("R", StringComparison.CurrentCultureIgnoreCase))
				{
					this.queryEditTitle.Visibility = (this.queryEditValue.Visibility = Visibility.Visible);
					this.editTitle.Visibility = (this.editValue.Visibility = (this.queryTitle.Visibility = (this.queryValue.Visibility = Visibility.Collapsed)));
					return;
				}
				this.queryEditTitle.Visibility = (this.queryEditValue.Visibility = Visibility.Collapsed);
				this.editTitle.Visibility = (this.editValue.Visibility = (this.queryTitle.Visibility = (this.queryValue.Visibility = Visibility.Visible)));
			}
		}

		// Token: 0x0600002E RID: 46 RVA: 0x00003EF4 File Offset: 0x000020F4
		private void showActionUIAndBindingDataContext()
		{
			if (this.CurrentSpecNode == null)
			{
				return;
			}
			if (this.CurrentSpecNode.SpecAction == null)
			{
				return;
			}
			if (this.CurrentSpecNode.SpecAction.IsActionDefaults)
			{
				return;
			}
			this.ActionSpecGroup.Visibility = Visibility.Visible;
			this.ActionSpecGroup.DataContext = this.CurrentSpecNode.SpecAction;
		}

		// Token: 0x0600002F RID: 47 RVA: 0x00003F50 File Offset: 0x00002150
		private void showActionSpecGroupDataContext(SpecActionNode node)
		{
			if (node == null)
			{
				return;
			}
			SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(node.ProgramKey).SpecificationInfo;
			if (specificationInfo == null)
			{
				return;
			}
			if (specificationInfo.IsActionDefault(node.Name))
			{
				this.ToolBarActionSpecGroup.DataContext = node;
				this.ToolBarActionSpecGroup.Visibility = Visibility.Visible;
				return;
			}
			this.ActionSpecGroup.DataContext = node;
			this.ActionSpecGroup.Visibility = Visibility.Visible;
		}

		// Token: 0x06000030 RID: 48 RVA: 0x00003FBC File Offset: 0x000021BC
		private void showProgRelFieldGroup()
		{
			if (this.CurrentSpecNode == null)
			{
				return;
			}
			if (this.CurrentSpecNode.SpecProgRel == null)
			{
				return;
			}
			SpecificationInfo specInfo = this.GetSpecInfo();
			if (specInfo == null)
			{
				return;
			}
			this.relDFComboBox.ItemsSource = specInfo.GetDependFieldsSource(this.CurrentSpecNode.GeneroComponent).ToList<string>();
			Binding binding = new Binding("Name");
			binding.Source = this.CurrentSpecNode.SpecProgRel;
			Binding binding2 = new Binding("Type");
			binding2.Source = this.CurrentSpecNode.GeneroComponent;
			Binding binding3 = new Binding("Parent.Type");
			binding3.Source = this.CurrentSpecNode.GeneroComponent;
			Binding binding4 = new Binding("Settings.StandardView");
			binding4.Source = PreferenceManager.Current;
			Binding binding5 = new Binding("[style]");
			binding5.Source = this.CurrentSpecNode.GeneroComponent;
			MultiBinding multiBinding = new MultiBinding();
			multiBinding.Converter = new MultiVisibilityConverter();
			multiBinding.ConverterParameter = "ProgRelFieldGroup";
			multiBinding.Bindings.Add(binding);
			multiBinding.Bindings.Add(binding2);
			multiBinding.Bindings.Add(binding3);
			multiBinding.Bindings.Add(binding4);
			multiBinding.Bindings.Add(binding5);
			this.ProgRelFieldGroup.SetBinding(UIElement.VisibilityProperty, multiBinding);
			this.ProgRelFieldGroup.DataContext = this.CurrentSpecNode.SpecProgRel;
		}

		// Token: 0x06000031 RID: 49 RVA: 0x00004124 File Offset: 0x00002324
		private void showReferenceFieldUIandBidningDataContext()
		{
			if (this.CurrentSpecNode == null)
			{
				return;
			}
			if (this.CurrentSpecNode.SpecReference == null)
			{
				return;
			}
			SpecificationInfo specInfo = this.GetSpecInfo();
			if (specInfo == null)
			{
				return;
			}
			this.refDFComboBox.ItemsSource = specInfo.GetDependFieldsSource(this.CurrentSpecNode.GeneroComponent).ToList<string>();
			this.RefFieldSpecGroup.Visibility = Visibility.Visible;
			this.RefFieldSpecGroup.DataContext = this.CurrentSpecNode.SpecReference;
			if (!this.CurrentSpecNode.Name.EndsWith("_desc"))
			{
				string text = Application.Current.FindResource("Message_InvalidReferenceFieldNameSuffix") as string;
				DesignerMessageBox.Show(text);
			}
		}

		// Token: 0x06000032 RID: 50 RVA: 0x000041C9 File Offset: 0x000023C9
		private void showHelpCodeBidningDataContext()
		{
			if (this.CurrentSpecNode == null)
			{
				return;
			}
			if (this.CurrentSpecNode.SpecHelpCode == null)
			{
				return;
			}
			this.HelpCodeSpecGroup.DataContext = this.CurrentSpecNode.SpecHelpCode;
		}

		// Token: 0x06000033 RID: 51 RVA: 0x000041F8 File Offset: 0x000023F8
		public static bool ValidMappingWidgetMixValue()
		{
			bool flag = true;
			if (SpecPropertyEditor.This != null && SpecPropertyEditor.This.CurrentSpecNode != null && SpecPropertyEditor.This.CurrentSpecNode.SpecHelpCode != null)
			{
				flag = SpecPropertyEditor.This.CurrentSpecNode.SpecHelpCode.ValidMappingWidgetMixValue();
				if (!flag)
				{
					string text = Application.Current.FindResource("specProperty_CountDiffMappingWidgetAndTableField") as string;
					DesignerMessageBox.Show(text);
				}
			}
			return flag;
		}

		// Token: 0x06000034 RID: 52 RVA: 0x000042C4 File Offset: 0x000024C4
		private void SRAttributesCB_Click(object sender, RoutedEventArgs e)
		{
			CheckBox checkBox = sender as CheckBox;
			bool flag = checkBox.IsChecked != null && checkBox.IsChecked.Value;
			string text = checkBox.Tag.ToString();
			XmlElement form = this.FormonlySpecGroup.DataContext as XmlElement;
			if (form == null)
			{
				return;
			}
			XElement xelement = (from s in SettingManager.Get().GetTzpManger(form.Key).SpecificationInfo.AssociateTable.Source.Descendants("sr")
				where s.Attribute("name").Value == form.Name && s.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)
				select s).FirstOrDefault<XElement>();
			if (xelement == null)
			{
				return;
			}
			string text2;
			if ((text2 = text) != null)
			{
				if (!(text2 == "insert"))
				{
					if (!(text2 == "delete"))
					{
						if (text2 == "append")
						{
							xelement.SetAttributeValue("append", flag ? "Y" : "N");
						}
					}
					else
					{
						xelement.SetAttributeValue("delete", flag ? "Y" : "N");
					}
				}
				else
				{
					xelement.SetAttributeValue("insert", flag ? "Y" : "N");
				}
			}
			xelement.SetAttributeValue("status", ReflectionHelpers.GetCustomDescription(SpecStatus.MODIFY));
			form.OnPropertyChanged("");
		}

		// Token: 0x06000035 RID: 53 RVA: 0x000044A0 File Offset: 0x000026A0
		private void showTableAssociationComboBox()
		{
			if (this.CurrentSpecNode == null)
			{
				return;
			}
			switch (this.CurrentSpecNode.Type)
			{
			case ComponentType.ScrollGrid:
			case ComponentType.Table:
			{
				this.InitTableAssociationSource(this.tableAssocCB);
				XElement xelement = (from s in SettingManager.Get().GetTzpManger(this.CurrentSpecNode.Key).SpecificationInfo.AssociateTable.Source.Descendants("sr")
					where s.Attribute("name").Value == this.CurrentSpecNode.Name && s.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)
					select s).FirstOrDefault<XElement>();
				if (xelement != null)
				{
					this.srInsertRowCB.IsChecked = new bool?(xelement.Attribute("insert") != null && xelement.Attribute("insert").Value == "Y");
					this.srDeleteRowCB.IsChecked = new bool?(xelement.Attribute("delete") != null && xelement.Attribute("delete").Value == "Y");
					this.srAppendRowCB.IsChecked = new bool?(xelement.Attribute("append") != null && xelement.Attribute("append").Value == "Y");
					return;
				}
				this.srInsertRowCB.IsChecked = new bool?(false);
				this.srDeleteRowCB.IsChecked = new bool?(false);
				this.srAppendRowCB.IsChecked = new bool?(false);
				return;
			}
			default:
				return;
			}
		}

		// Token: 0x06000036 RID: 54 RVA: 0x00004638 File Offset: 0x00002838
		private void InitTableAssociationSource(ComboBox cbo)
		{
			cbo.SelectionChanged -= this.tableAssocCB_SelectionChanged;
			cbo.DataContext = null;
			SpecificationInfo specInfo = this.GetSpecInfo();
			if (specInfo == null)
			{
				return;
			}
			IEnumerable<XElement> enumerable = specInfo.AssociateTable.Source.Elements("tbl");
			XElement xelement = null;
			List<string> list = new List<string>();
			foreach (XElement xelement2 in enumerable)
			{
				if (!(xelement2.Attribute("status").Value == ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)))
				{
					list.Add(xelement2.Attribute("name").Value);
					if (xelement == null)
					{
						foreach (XElement xelement3 in xelement2.Elements("sr"))
						{
							if (xelement3.Attribute("name").Value == this.CurrentSpecNode.Name && xelement3.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE))
							{
								xelement = xelement3;
							}
						}
					}
				}
			}
			cbo.ItemsSource = list;
			BindingExpression bindingExpression = cbo.GetBindingExpression(ComboBox.TextProperty);
			if (bindingExpression != null)
			{
				Validation.ClearInvalid(bindingExpression);
			}
			if (xelement != null)
			{
				cbo.DataContext = xelement.Parent.Attribute("name").Value;
			}
			else if (bindingExpression != null)
			{
				Validation.MarkInvalid(bindingExpression, new ValidationError(new ExceptionValidationRule(), bindingExpression)
				{
					ErrorContent = string.Format(Application.Current.FindResource("Message_NotSetTableAssociation") as string, this.CurrentSpecNode.Name)
				});
			}
			cbo.SelectionChanged += this.tableAssocCB_SelectionChanged;
		}

		// Token: 0x06000037 RID: 55 RVA: 0x0000494C File Offset: 0x00002B4C
		private void tableAssocCB_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			if (e.AddedItems.Count == 0)
			{
				return;
			}
			if (this.handledTblAssocChanged)
			{
				ComboBox comboBox = sender as ComboBox;
				SpecificationInfo specInfo = this.GetSpecInfo();
				if (specInfo == null)
				{
					return;
				}
				XElement source = specInfo.AssociateTable.Source;
				string srName = this.CurrentSpecNode.Name;
				string oldTblName = comboBox.Text;
				string newTblName = e.AddedItems[0].ToString();
				XElement xelement = (from t in source.Elements("tbl")
					where t.Attribute("name").Value == oldTblName && t.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)
					select t).FirstOrDefault<XElement>();
				XElement xelement2 = (from t in source.Elements("tbl")
					where t.Attribute("name").Value == newTblName && t.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)
					select t).First<XElement>();
				if (xelement2 != null)
				{
					if (xelement != null)
					{
						XElement xelement3 = (from s in xelement.Elements("sr")
							where s.Attribute("name").Value == srName
							select s).First<XElement>();
						xelement3.SetAttributeValue("status", ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE));
					}
					(from s in xelement2.Elements("sr")
						where s.Attribute("name").Value == srName
						select s).Remove<XElement>();
					XElement xelement4 = XElement.Parse("<sr name='' src='s' status='u' cascade='Y' />", LoadOptions.None);
					xelement4.Add(new XAttribute("insert", "Y"));
					xelement4.Add(new XAttribute("append", "Y"));
					xelement4.Add(new XAttribute("delete", "Y"));
					xelement4.SetAttributeValue("name", srName);
					xelement4.SetAttributeValue("src", specInfo.Env);
					xelement4.SetAttributeValue("status", ReflectionHelpers.GetCustomDescription(SpecStatus.MODIFY));
					xelement4.SetAttributeValue("kind", (this.CurrentSpecNode.Type != ComponentType.Unknown) ? this.CurrentSpecNode.Type.ToString() : "");
					xelement2.Add(xelement4);
					this.srInsertRowCB.IsChecked = new bool?(true);
					this.srDeleteRowCB.IsChecked = new bool?(true);
					this.srAppendRowCB.IsChecked = new bool?(true);
				}
			}
			this.handledTblAssocChanged = true;
		}

		// Token: 0x06000038 RID: 56 RVA: 0x00004BD4 File Offset: 0x00002DD4
		private void cb_tables_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			ComboBox comboBox = sender as ComboBox;
			if (comboBox.SelectedIndex == -1)
			{
				return;
			}
			if (!(this.FieldSpecGroup1.DataContext is SpecFieldNode))
			{
				return;
			}
			string text = comboBox.SelectedItem as string;
			this.cb_table_name.Text = TableColumnHelper.GetTableDesc(text);
		}

		// Token: 0x06000039 RID: 57 RVA: 0x00004C24 File Offset: 0x00002E24
		private void cb_fields_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			ComboBox comboBox = sender as ComboBox;
			string text = comboBox.SelectedItem as string;
			SpecFieldNode specFieldNode = this.FieldSpecGroup1.DataContext as SpecFieldNode;
			if (string.IsNullOrEmpty(text) || specFieldNode == null)
			{
				this.column_name.Text = (this.column_type.Text = (this.column_type2.Text = string.Empty));
				return;
			}
			XElement columnInfo = TableColumnHelper.GetColumnInfo(specFieldNode.Table, text);
			if (columnInfo == null)
			{
				this.column_name.Text = (this.column_type.Text = (this.column_type2.Text = string.Empty));
				return;
			}
			this.column_name.Text = ((columnInfo.Attribute("text") != null) ? columnInfo.Attribute("text").Value : string.Empty);
			this.column_type.Text = ((columnInfo.Attribute("type") != null) ? columnInfo.Attribute("type").Value : string.Empty);
			this.column_type2.Text = ((columnInfo.Attribute("attribute") != null) ? columnInfo.Attribute("attribute").Value : string.Empty);
		}

		// Token: 0x0600003A RID: 58 RVA: 0x00004D94 File Offset: 0x00002F94
		public static void ShowZoomsWindow(TextBox buttonEdit)
		{
			SpecDesigner.FormDataEditor.SelectionMode selectionMode = SpecDesigner.FormDataEditor.SelectionMode.Single;
			string text = "";
			SpecDesigner.FormDataEditor.MasterDetailView masterDetailView = null;
			string text2;
			if ((text2 = buttonEdit.Tag.ToString()) != null)
			{
				if (!(text2 == "i_zoom"))
				{
					if (!(text2 == "c_zoom"))
					{
						if (!(text2 == "chk_ref"))
						{
							if (!(text2 == "prog_rel"))
							{
								if (text2 == "items")
								{
									text = (string)Application.Current.FindResource("specProperty_items");
									string text3 = SettingManager.Get().Info_Items;
									masterDetailView = new SpecDesigner.FormDataEditor.MasterDetailView(text3, buttonEdit.Text);
								}
							}
							else
							{
								string text3 = SettingManager.Get().Info_ProgRel;
								text = (string)Application.Current.FindResource("specProperty_prog_rel");
								masterDetailView = new SpecDesigner.FormDataEditor.MasterDetailView(text3, selectionMode, buttonEdit.Text);
							}
						}
						else
						{
							string text3 = SettingManager.Get().Info_Checks;
							text = (string)Application.Current.FindResource("specProperty_chk_ref");
							masterDetailView = new SpecDesigner.FormDataEditor.MasterDetailView(text3, selectionMode, buttonEdit.Text);
						}
					}
					else
					{
						string text3 = SettingManager.Get().Info_Zooms;
						text = (string)Application.Current.FindResource("specProperty_c_zoom");
						masterDetailView = new SpecDesigner.FormDataEditor.MasterDetailView(text3, selectionMode, buttonEdit.Text);
					}
				}
				else
				{
					string text3 = SettingManager.Get().Info_Zooms;
					text = (string)Application.Current.FindResource("specProperty_i_zoom");
					masterDetailView = new SpecDesigner.FormDataEditor.MasterDetailView(text3, selectionMode, buttonEdit.Text);
				}
			}
			Window window = new Window
			{
				Title = text,
				Content = masterDetailView
			};
			masterDetailView.ItemSelected += delegate(object s1, RoutedEventArgs e1)
			{
				window.Close();
			};
			window.ShowDialog();
			if (!buttonEdit.IsReadOnly && !string.IsNullOrEmpty(masterDetailView.SelectedID))
			{
				buttonEdit.Text = masterDetailView.SelectedID;
				Keyboard.Focus(buttonEdit);
			}
		}

		// Token: 0x0600003B RID: 59 RVA: 0x00004F7A File Offset: 0x0000317A
		private void actionNameTB_PreviewTextInput(object sender, TextCompositionEventArgs e)
		{
			if (Regex.IsMatch(e.Text, "\\W|[ ]"))
			{
				e.Handled = true;
			}
		}

		// Token: 0x0600003C RID: 60 RVA: 0x00004F95 File Offset: 0x00003195
		private void actionNameTB_PreviewKeyDown(object sender, KeyEventArgs e)
		{
			if (e.Key == Key.Space)
			{
				e.Handled = true;
			}
		}

		// Token: 0x0600003D RID: 61 RVA: 0x00004FA8 File Offset: 0x000031A8
		private void excludeViewBtn_Click(object sender, RoutedEventArgs e)
		{
			SpecExcludeView.ShowDialog(this.CurrentSpecNode.Key);
		}

		// Token: 0x0600003E RID: 62 RVA: 0x00004FBA File Offset: 0x000031BA
		public void CanShowItemsWindow(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = e.Parameter is TextBox;
		}

		// Token: 0x0600003F RID: 63 RVA: 0x00004FD4 File Offset: 0x000031D4
		public void ExecutedShowItemsWindow(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			TextBox textBox = e.Parameter as TextBox;
			if (textBox == null)
			{
				return;
			}
			XmlElement xmlElement = textBox.DataContext as XmlElement;
			new ItemsPropertyEditor
			{
				ComponentName = xmlElement.Name,
				Widget = xmlElement.GetAttribute("widget"),
				ProgramKey = this.CurrentSpecNode.Key,
				DataContext = textBox.DataContext
			}.ShowDialog();
		}

		// Token: 0x06000040 RID: 64 RVA: 0x0000504B File Offset: 0x0000324B
		public void CanShowLocalStringWindow(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = e.Parameter is TextBox;
		}

		// Token: 0x06000041 RID: 65 RVA: 0x000050B8 File Offset: 0x000032B8
		public void ExecutedShowLocalStringWindow(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			TextBox textBox = e.Parameter as TextBox;
			SpecNodeType specNodeType = ((this.CurrentSpecNode != null && this.CurrentSpecNode.GeneroComponent == null) ? SpecNodeType.ACTION : SpecNodeType.FIELD);
			SpecFieldTextControl specTextContent = new SpecFieldTextControl(this.CurrentSpecNode.Key, textBox.Text, specNodeType);
			AbstractStringNode selectedStringNode = null;
			Window textWindow = null;
			specTextContent.TextIDSelected += delegate(object s, AbstractStringNode ea)
			{
				selectedStringNode = ea;
				textWindow.Close();
			};
			string text = Application.Current.FindResource("WinTitle_TextStringSetting").ToString();
			textWindow = new Window
			{
				Width = 400.0,
				Height = 350.0,
				Title = text,
				Content = specTextContent
			};
			textWindow.Closing += delegate(object s, CancelEventArgs e1)
			{
				specTextContent.dataGrid.CommitEdit();
				specTextContent.dataGrid.CancelEdit();
				specTextContent.dataGrid.DataContext = null;
			};
			textWindow.ShowDialog();
			if (selectedStringNode != null)
			{
				textBox.Text = selectedStringNode.Name;
			}
			BindingExpression bindingExpression = textBox.GetBindingExpression(TextBox.TextProperty);
			bindingExpression.UpdateSource();
		}

		// Token: 0x14000001 RID: 1
		// (add) Token: 0x06000042 RID: 66 RVA: 0x000051F0 File Offset: 0x000033F0
		// (remove) Token: 0x06000043 RID: 67 RVA: 0x00005228 File Offset: 0x00003428
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x06000044 RID: 68 RVA: 0x0000525D File Offset: 0x0000345D
		private void OnPropertyChanged(string propertyName)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(propertyName));
			}
		}

		// Token: 0x06000045 RID: 69 RVA: 0x0000527C File Offset: 0x0000347C
		private void CheckBox_Checked(object sender, RoutedEventArgs e)
		{
			if (this.CurrentSpecNode == null)
			{
				return;
			}
			if (this.cb_fields.SelectedItem == null)
			{
				return;
			}
			CheckBox checkBox = sender as CheckBox;
			string text = this.cb_fields.SelectedItem.ToString();
			SpecFieldNode specFieldNode = this.FieldSpecGroup1.DataContext as SpecFieldNode;
			if (string.IsNullOrEmpty(text) || specFieldNode == null)
			{
				return;
			}
			XElement columnInfo = TableColumnHelper.GetColumnInfo(specFieldNode.Table, text);
			if (!checkBox.IsChecked.Value)
			{
				if (columnInfo.Attribute("pk").Value.Equals("Y"))
				{
					if (MessageBoxResult.OK == DesignerMessageBox.Show(Application.Current.FindResource("Message_SpecPKNotNull") as string, this.CurrentSpecNode.Key.Program, MessageBoxButton.OKCancel, MessageBoxImage.Asterisk))
					{
						checkBox.IsChecked = new bool?(false);
						return;
					}
					checkBox.IsChecked = new bool?(true);
					return;
				}
				else if (columnInfo.Attribute("req").Value.Equals("Y"))
				{
					if (MessageBoxResult.OK == DesignerMessageBox.Show(Application.Current.FindResource("Message_SpecReqNotNull") as string, this.CurrentSpecNode.Key.Program, MessageBoxButton.OKCancel, MessageBoxImage.Asterisk))
					{
						checkBox.IsChecked = new bool?(false);
						return;
					}
					checkBox.IsChecked = new bool?(true);
				}
			}
		}

		// Token: 0x06000046 RID: 70 RVA: 0x000053C9 File Offset: 0x000035C9
		private void PropertyItem_notNull_CheckBox_Checked(object sender, RoutedEventArgs e)
		{
		}

		// Token: 0x06000047 RID: 71 RVA: 0x000053CB File Offset: 0x000035CB
		private void PropertyItem_notNull_CheckBox_Unchecked(object sender, RoutedEventArgs e)
		{
		}

		// Token: 0x06000048 RID: 72 RVA: 0x000053CD File Offset: 0x000035CD
		private void cb_fields_LostFocus(object sender, RoutedEventArgs e)
		{
		}

		// Token: 0x04000007 RID: 7
		public static Dictionary<string, Dictionary<string, string>> ComponentPropertyVisibilityMap = new Dictionary<string, Dictionary<string, string>>();

		// Token: 0x04000008 RID: 8
		private FormSpecModel _currentSpecNode;

		// Token: 0x04000009 RID: 9
		private Dictionary<string, IEnumerable<string>> LastSpecArgs = new Dictionary<string, IEnumerable<string>>();

		// Token: 0x0400000A RID: 10
		private static SpecPropertyEditor _this = new SpecPropertyEditor();

		// Token: 0x0400000B RID: 11
		private PackageKey _programKey;

		// Token: 0x0400000C RID: 12
		private bool handledTblAssocChanged = true;
	}
}
