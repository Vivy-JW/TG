using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Markup;
using System.Xml.Linq;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Views
{
	// Token: 0x0200003B RID: 59
	public partial class ToolBar : UserControl, IDisposable
	{
		// Token: 0x06000214 RID: 532 RVA: 0x0000B5C2 File Offset: 0x000097C2
		public ToolBar()
		{
			this.InitializeComponent();
			base.DataContextChanged += this.ToolBar_DataContextChanged;
		}

		// Token: 0x06000215 RID: 533 RVA: 0x0000B5E4 File Offset: 0x000097E4
		private void ToolBar_DataContextChanged(object sender, DependencyPropertyChangedEventArgs e)
		{
			SpecificationInfo specificationInfo = e.NewValue as SpecificationInfo;
			if (specificationInfo == null)
			{
				return;
			}
			this.ProgramKey = specificationInfo.Key;
			base.DataContextChanged -= this.ToolBar_DataContextChanged;
			this.LoadToolBar(specificationInfo.Key);
		}

		// Token: 0x06000216 RID: 534 RVA: 0x0000B62C File Offset: 0x0000982C
		private void LoadToolBar(PackageKey progKey)
		{
			this.ProgramKey = progKey;
			string @class = SettingManager.Get().GetTzpManger(this.ProgramKey).SpecificationInfo.GetClass();
			string text = SettingManager.Get().LoadToolBar(@class);
			if (!string.IsNullOrEmpty(text))
			{
				this.toolbarLoaded(XDocument.Parse(text));
				return;
			}
			base.Visibility = Visibility.Collapsed;
		}

		// Token: 0x06000217 RID: 535 RVA: 0x0000B684 File Offset: 0x00009884
		public void toolbarLoaded(XDocument toolbar)
		{
			this.itemsContainer.Items.Clear();
			SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(this.ProgramKey).SpecificationInfo;
			IEnumerable<XElement> enumerable = toolbar.Element("ToolBar").Elements();
			foreach (XElement xelement in enumerable)
			{
				if ("ToolBarSeparator".Equals(xelement.Name.LocalName))
				{
					this.itemsContainer.Items.Add(new Separator());
				}
				else
				{
					string value = xelement.Attribute("name").Value;
					ToolBarItem toolBarItemImage = this.GetToolBarItemImage(value);
					if (toolBarItemImage != null && toolBarItemImage.Model != null)
					{
						this.itemsContainer.Items.Add(toolBarItemImage);
						if (specificationInfo.AllowedAction == null)
						{
							toolBarItemImage.IsEnable = false;
						}
						else if (specificationInfo.AllowedAction != null && !specificationInfo.AllowedAction.Contains(value))
						{
							toolBarItemImage.IsEnable = false;
						}
						else
						{
							SpecificationInfo specificationInfo2 = SettingManager.Get().GetTzpManger(this.ProgramKey).SpecificationInfo;
							SpecActionNode specActionNode = specificationInfo2.FindActionDefaultSpecOrCreate(value);
							toolBarItemImage.DataContext = specActionNode;
							toolBarItemImage.IsEnable = true;
							toolBarItemImage.ToolItemClicked += this.item_ToolItemClicked;
						}
					}
				}
			}
		}

		// Token: 0x06000218 RID: 536 RVA: 0x0000B80C File Offset: 0x00009A0C
		private void item_ToolItemClicked(object sender, string actionName)
		{
			SpecArgs specArgs = new SpecArgs(actionName, ComponentType.Unknown, this.ProgramKey);
			EventAggregatorManager.Global.GetEvent<ComponentSelectedEvent>().Publish(specArgs);
		}

		// Token: 0x06000219 RID: 537 RVA: 0x0000B8B8 File Offset: 0x00009AB8
		private ToolBarItem GetToolBarItemImage(string name)
		{
			string text = name;
			XElement xelement;
			if (SettingManager.Get().GetTzpManger(this.ProgramKey).ActionDefaults != null)
			{
				xelement = (from ad in SettingManager.Get().GetTzpManger(this.ProgramKey).ActionDefaults.Elements()
					where name.Equals(ad.Attribute("name").Value) && (ad.Attribute("defaultView") == null || (ad.Attribute("defaultView") != null && "yes".Equals(ad.Attribute("defaultView").Value)))
					select ad).ElementAtOrDefault<XElement>(0);
				if (xelement == null)
				{
					return null;
				}
				text = xelement.Attribute("text").Value;
			}
			else
			{
				xelement = new XElement("ActionDefault", new object[]
				{
					new XAttribute("name", name),
					new XAttribute("image", ""),
					new XAttribute("text", name),
					new XAttribute("defaultView", "yes")
				});
			}
			return new ToolBarItem(xelement)
			{
				ItemName = name,
				Text = text
			};
		}

		// Token: 0x17000068 RID: 104
		// (get) Token: 0x0600021A RID: 538 RVA: 0x0000B9E8 File Offset: 0x00009BE8
		public ItemCollection ToolBarItems
		{
			get
			{
				return this.itemsContainer.Items;
			}
		}

		// Token: 0x0600021B RID: 539 RVA: 0x0000B9F8 File Offset: 0x00009BF8
		public void Dispose()
		{
			foreach (ToolBarItem toolBarItem in this.itemsContainer.Items.OfType<ToolBarItem>())
			{
				toolBarItem.ToolItemClicked -= this.item_ToolItemClicked;
				BindingOperations.ClearAllBindings(toolBarItem);
			}
			this.itemsContainer.Items.Clear();
		}

		// Token: 0x04000127 RID: 295
		private PackageKey ProgramKey;

		// Token: 0x04000128 RID: 296
		public static readonly DependencyProperty ToolBarItemsProperty = DependencyProperty.Register("ToolBarItems", typeof(ItemCollection), typeof(WidgetBox), null);
	}
}
