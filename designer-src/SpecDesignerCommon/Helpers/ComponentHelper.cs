using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Linq;
using System.Windows.Input;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.ViewModel;

namespace SpecDesignerCommon.Helpers
{
	// Token: 0x02000118 RID: 280
	public class ComponentHelper
	{
		// Token: 0x060009F4 RID: 2548 RVA: 0x00031A20 File Offset: 0x0002FC20
		public static ComponentHelper Get(PackageKey key)
		{
			ComponentHelper componentHelper = null;
			if (!ComponentHelper._services.TryGetValue(key, out componentHelper))
			{
				componentHelper = new ComponentHelper(key);
				ComponentHelper._services.Add(key, componentHelper);
			}
			return componentHelper;
		}

		// Token: 0x060009F5 RID: 2549 RVA: 0x00031A52 File Offset: 0x0002FC52
		public ComponentHelper(PackageKey key)
		{
			this._key = key;
		}

		// Token: 0x170002A0 RID: 672
		// (get) Token: 0x060009F6 RID: 2550 RVA: 0x00031BFC File Offset: 0x0002FDFC
		public IEnumerable<XmlElement> SelectedObjects
		{
			get
			{
				foreach (XmlElement e in this._selected)
				{
					yield return e;
				}
				yield break;
			}
		}

		// Token: 0x060009F7 RID: 2551 RVA: 0x00031C19 File Offset: 0x0002FE19
		public void Move(MoveDirection direction, int offset)
		{
			FormCommands.MoveCommand.Execute(direction, null);
		}

		// Token: 0x060009F8 RID: 2552 RVA: 0x00031C2C File Offset: 0x0002FE2C
		public void DeleteSelection()
		{
			ApplicationCommands.Delete.Execute(null, null);
		}

		// Token: 0x060009F9 RID: 2553 RVA: 0x00031C70 File Offset: 0x0002FE70
		private void AddObjectIntoSelected(XmlElement source)
		{
			this._selected.Add(source);
			this._selected.Sort(delegate(XmlElement a, XmlElement b)
			{
				int num;
				try
				{
					num = a.Index - b.Index;
				}
				catch
				{
					num = 0;
				}
				return num;
			});
		}

		// Token: 0x060009FA RID: 2554 RVA: 0x00031CA8 File Offset: 0x0002FEA8
		public void AddSelection(XmlElement source, bool apppendSelection)
		{
			if (!apppendSelection)
			{
				this.ClearSelection();
			}
			if (source == null)
			{
				return;
			}
			if (this._selected.Count > 0 && source.Parent != this._selected[0].Parent)
			{
				return;
			}
			source.IsSelected = !source.IsSelected;
			if (!source.IsSelected)
			{
				this._selected.Remove(source);
				return;
			}
			this.AddObjectIntoSelected(source);
			if (1 == this._selected.Count)
			{
				SpecArgs specArgs = new SpecArgs(source.Name, source.Type, source.Key);
				EventAggregatorManager.Global.GetEvent<ComponentSelectedEvent>().Publish(specArgs);
				return;
			}
			List<string> list = new List<string>();
			foreach (XmlElement xmlElement in this._selected)
			{
				list.Add(xmlElement.Name);
			}
			MultiSelectionArgs multiSelectionArgs = new MultiSelectionArgs(source.Key, list);
			EventAggregatorManager.Global.GetEvent<MultiComponentSelectedEvent>().Publish(multiSelectionArgs);
		}

		// Token: 0x060009FB RID: 2555 RVA: 0x00031DC0 File Offset: 0x0002FFC0
		public void SelectAction(string actionName, PackageKey key)
		{
			this.ClearSelection();
			SpecArgs specArgs = new SpecArgs(actionName, ComponentType.Unknown, key);
			EventAggregatorManager.Global.GetEvent<ComponentSelectedEvent>().Publish(specArgs);
		}

		// Token: 0x060009FC RID: 2556 RVA: 0x00031DEC File Offset: 0x0002FFEC
		public void MultipleSelection(IEnumerable<XmlElement> elements)
		{
			this.ClearSelection();
			List<string> list = new List<string>();
			PackageKey packageKey = null;
			foreach (XmlElement xmlElement in elements)
			{
				if (null == packageKey)
				{
					packageKey = xmlElement.Key;
				}
				xmlElement.IsSelected = true;
				this.AddObjectIntoSelected(xmlElement);
				list.Add(xmlElement.Name);
			}
			if (elements.Count<XmlElement>() == 1)
			{
				XmlElement xmlElement2 = elements.First<XmlElement>();
				SpecArgs specArgs = new SpecArgs(xmlElement2.Name, xmlElement2.Type, xmlElement2.Key);
				EventAggregatorManager.Global.GetEvent<ComponentSelectedEvent>().Publish(specArgs);
				return;
			}
			if (elements.Count<XmlElement>() > 1)
			{
				MultiSelectionArgs multiSelectionArgs = new MultiSelectionArgs(packageKey, list);
				EventAggregatorManager.Global.GetEvent<MultiComponentSelectedEvent>().Publish(multiSelectionArgs);
			}
		}

		// Token: 0x060009FD RID: 2557 RVA: 0x00031EC8 File Offset: 0x000300C8
		private void source_PropertyChanged(object sender, PropertyChangedEventArgs e)
		{
			string text;
			if ((text = e.PropertyName.ToLower()) != null && !(text == "Height"))
			{
				text == "Width";
			}
		}

		// Token: 0x060009FE RID: 2558 RVA: 0x00031F00 File Offset: 0x00030100
		public void ClearSelection()
		{
			foreach (XmlElement xmlElement in this._selected)
			{
				xmlElement.IsSelected = false;
			}
			this._selected.Clear();
		}

		// Token: 0x060009FF RID: 2559 RVA: 0x00031F60 File Offset: 0x00030160
		public bool SelectionContains(XmlElement target)
		{
			foreach (XmlElement xmlElement in this._selected)
			{
				if (xmlElement == target)
				{
					return true;
				}
			}
			return false;
		}

		// Token: 0x170002A1 RID: 673
		// (get) Token: 0x06000A00 RID: 2560 RVA: 0x00031FB8 File Offset: 0x000301B8
		// (set) Token: 0x06000A01 RID: 2561 RVA: 0x00031FC0 File Offset: 0x000301C0
		public bool IsSizeChanging { get; set; }

		// Token: 0x040003BD RID: 957
		private static Dictionary<PackageKey, ComponentHelper> _services = new Dictionary<PackageKey, ComponentHelper>();

		// Token: 0x040003BE RID: 958
		private PackageKey _key;

		// Token: 0x040003BF RID: 959
		private List<XmlElement> _selected = new List<XmlElement>();
	}
}
