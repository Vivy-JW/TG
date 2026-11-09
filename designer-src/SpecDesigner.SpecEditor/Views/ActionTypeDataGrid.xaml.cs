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
using System.Windows.Markup;
using System.Xml.Linq;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Views
{
	// Token: 0x02000014 RID: 20
	public partial class ActionTypeDataGrid : UserControl
	{
		// Token: 0x0600007D RID: 125 RVA: 0x000067DE File Offset: 0x000049DE
		public ActionTypeDataGrid()
		{
			this.InitializeComponent();
			base.DataContextChanged += this.ActionTypeDataGrid_DataContextChanged;
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Subscribe(new Action<PackageKey>(this.Subscribe_TzpClosed));
		}

		// Token: 0x0600007E RID: 126 RVA: 0x0000681C File Offset: 0x00004A1C
		private void ActionTypeDataGrid_DataContextChanged(object sender, DependencyPropertyChangedEventArgs e)
		{
			if (e.NewValue is SpecActionNode)
			{
				SpecActionNode specActionNode = e.NewValue as SpecActionNode;
				this._programKey = specActionNode.ProgramKey;
				this.RenderTypesToDataGrid(specActionNode);
				EventAggregatorManager.Get(specActionNode.ProgramKey).GetEvent<SpecPropertiesChangedEvent>().Unsubscribe(new Action<PackageKey>(this.Subscribe_SpecPropertyChanged));
				EventAggregatorManager.Get(specActionNode.ProgramKey).GetEvent<SpecPropertiesChangedEvent>().Subscribe(new Action<PackageKey>(this.Subscribe_SpecPropertyChanged));
				return;
			}
			if (this._programKey != null)
			{
				try
				{
					EventAggregatorManager.Get(this._programKey).GetEvent<SpecPropertiesChangedEvent>().Unsubscribe(new Action<PackageKey>(this.Subscribe_SpecPropertyChanged));
				}
				catch
				{
				}
			}
			this.ClearActionTypeData();
			this._programKey = null;
		}

		// Token: 0x0600007F RID: 127 RVA: 0x000068EC File Offset: 0x00004AEC
		public void Subscribe_TzpClosed(PackageKey programKey)
		{
			try
			{
				EventAggregatorManager.Get(programKey).GetEvent<SpecPropertiesChangedEvent>().Unsubscribe(new Action<PackageKey>(this.Subscribe_SpecPropertyChanged));
				this.ClearActionTypeData();
				this._programKey = null;
			}
			catch
			{
			}
		}

		// Token: 0x06000080 RID: 128 RVA: 0x00006938 File Offset: 0x00004B38
		public void Subscribe_SpecPropertyChanged(PackageKey key)
		{
			if (this._programKey == key)
			{
				foreach (object obj in ((IEnumerable)this.dg.Items))
				{
					ActionTypeData actionTypeData = (ActionTypeData)obj;
					actionTypeData.ActionTypeChanged();
				}
			}
		}

		// Token: 0x06000081 RID: 129 RVA: 0x000069CC File Offset: 0x00004BCC
		private void RenderTypesToDataGrid(SpecActionNode action)
		{
			this.ClearActionTypeData();
			List<ActionTypeData> list = new List<ActionTypeData>();
			list.Add(new ActionTypeData(action, "all")
			{
				ActionDescription = (Application.Current.FindResource("specProperty_all") as string)
			});
			list.Add(new ActionTypeData(action, "mi")
			{
				ActionDescription = (Application.Current.FindResource("specProperty_mi") as string)
			});
			if (this._programKey != null)
			{
				XElement source = SettingManager.Get().GetTzpManger(this._programKey).SpecificationInfo.AssociateTable.Source;
				string text = "^s_detail(?<seq>\\d+)$";
				List<XElement> list2 = (from sr in source.Descendants("sr")
					where sr.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)
					select sr).ToList<XElement>();
				foreach (XElement xelement in list2)
				{
					string value = xelement.Attribute("name").Value;
					if (Regex.IsMatch(value, text))
					{
						string value2 = Regex.Matches(value, text)[0].Groups["seq"].Value;
						list.Add(new ActionTypeData(action, "di" + value2)
						{
							ActionDescription = value + (Application.Current.FindResource("specProperty_di") as string)
						});
						list.Add(new ActionTypeData(action, "db" + value2)
						{
							ActionDescription = value + (Application.Current.FindResource("specProperty_db") as string)
						});
					}
				}
				this.dg.ItemsSource = list;
			}
		}

		// Token: 0x06000082 RID: 130 RVA: 0x00006BD8 File Offset: 0x00004DD8
		public void ClearActionTypeData()
		{
			foreach (object obj in ((IEnumerable)this.dg.Items))
			{
				ActionTypeData actionTypeData = (ActionTypeData)obj;
				actionTypeData.Clear();
			}
			this.dg.ItemsSource = null;
		}

		// Token: 0x0400004F RID: 79
		private PackageKey _programKey;
	}
}
