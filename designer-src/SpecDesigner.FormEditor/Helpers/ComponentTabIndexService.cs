using System;
using System.Collections.Generic;
using System.Linq;
using System.Windows.Input;
using SpecDesignerCommon;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Helpers
{
	// Token: 0x02000031 RID: 49
	public class ComponentTabIndexService : IDisposable
	{
		// Token: 0x060001B5 RID: 437 RVA: 0x00008DCC File Offset: 0x00006FCC
		public static ComponentTabIndexService Get(PackageKey key)
		{
			ComponentTabIndexService componentTabIndexService = null;
			if (!ComponentTabIndexService._services.TryGetValue(key, out componentTabIndexService))
			{
				componentTabIndexService = new ComponentTabIndexService(key);
				ComponentTabIndexService._services.Add(key, componentTabIndexService);
			}
			return componentTabIndexService;
		}

		// Token: 0x060001B6 RID: 438 RVA: 0x00008DFE File Offset: 0x00006FFE
		public ComponentTabIndexService(PackageKey key)
		{
			this._key = key;
		}

		// Token: 0x060001B7 RID: 439 RVA: 0x00008E23 File Offset: 0x00007023
		internal void Register(XmlElement mainRoot)
		{
			this._mainRoot = mainRoot;
			this.iteratorAllNodes(this._mainRoot);
		}

		// Token: 0x060001B8 RID: 440 RVA: 0x00008E38 File Offset: 0x00007038
		internal void UnRegister()
		{
			ComponentTabIndexService._services.Remove(this._key);
		}

		// Token: 0x060001B9 RID: 441 RVA: 0x00008E4B File Offset: 0x0000704B
		internal void RemoveElement(XmlElement element)
		{
			this._sourceList.Remove(element);
			this._tabIndexedList.Remove(element);
		}

		// Token: 0x1700005F RID: 95
		// (get) Token: 0x060001BA RID: 442 RVA: 0x00008E67 File Offset: 0x00007067
		public int CurrentIndex
		{
			get
			{
				return this._tabIndexedList.Count<XmlElement>();
			}
		}

		// Token: 0x060001BB RID: 443 RVA: 0x00008E74 File Offset: 0x00007074
		public void Dispose()
		{
			this._sourceList.Clear();
			this._tabIndexedList.Clear();
		}

		// Token: 0x060001BC RID: 444 RVA: 0x00008E8C File Offset: 0x0000708C
		public void SetAsCurrent(XmlElement m)
		{
			this._sourceList.InsertRange(0, this._tabIndexedList);
			this._tabIndexedList.Clear();
			int num = this._sourceList.IndexOf(m);
			this._tabIndexedList.AddRange(this._sourceList.GetRange(0, num + 1));
			this._sourceList.RemoveRange(0, num + 1);
			this.ArrangeTabIndex();
		}

		// Token: 0x060001BD RID: 445 RVA: 0x00008EF2 File Offset: 0x000070F2
		public void SetAsFirst(XmlElement m)
		{
			this.RemoveElement(m);
			this._sourceList.InsertRange(0, this._tabIndexedList);
			this._tabIndexedList.Clear();
			this._tabIndexedList.Insert(0, m);
			this.ArrangeTabIndex();
		}

		// Token: 0x060001BE RID: 446 RVA: 0x00008F2C File Offset: 0x0000712C
		public void SetAsNext(XmlElement m)
		{
			if (this.CurrentIndex.ToString().Equals(m.GetAttribute("tabIndex")))
			{
				return;
			}
			if ((Keyboard.Modifiers & ModifierKeys.Control) == ModifierKeys.Control)
			{
				this._sourceList.InsertRange(0, this._tabIndexedList);
				this._tabIndexedList.Clear();
			}
			if (this._tabIndexedList.Count<XmlElement>() == 0)
			{
				if (string.IsNullOrEmpty(m.GetAttribute("tabIndex")))
				{
					this.RemoveElement(m);
					this._tabIndexedList.Add(m);
					m["tabIndex"] = "1";
				}
				else
				{
					int num = this._sourceList.IndexOf(m);
					this._tabIndexedList.AddRange(this._sourceList.GetRange(0, num + 1));
					this._sourceList.RemoveRange(0, num + 1);
				}
			}
			else
			{
				this.RemoveElement(m);
				this._tabIndexedList.Add(m);
			}
			this.ArrangeTabIndex();
		}

		// Token: 0x060001BF RID: 447 RVA: 0x00009016 File Offset: 0x00007216
		public void SetAsNonTabable(XmlElement m)
		{
			m["tabIndex"] = "";
			this.RemoveElement(m);
			this._sourceList.Add(m);
			this.ArrangeTabIndex();
		}

		// Token: 0x060001C0 RID: 448 RVA: 0x00009041 File Offset: 0x00007241
		public void ShiftCurrent(XmlElement m)
		{
			this.RemoveElement(m);
			this._tabIndexedList.Add(m);
			this.ArrangeTabIndex();
		}

		// Token: 0x060001C1 RID: 449 RVA: 0x0000905C File Offset: 0x0000725C
		public void SwapSelected(XmlElement m)
		{
			int num = this._tabIndexedList.IndexOf(m);
			XmlElement xmlElement = this._tabIndexedList.Last<XmlElement>();
			if (num != -1)
			{
				this._tabIndexedList.Add(m);
				this._tabIndexedList.Remove(xmlElement);
				this._tabIndexedList.RemoveAt(num);
				this._tabIndexedList.Insert(num, xmlElement);
			}
			else
			{
				num = this._sourceList.IndexOf(m);
				this._sourceList.Remove(m);
				this._sourceList.Insert(num, xmlElement);
				this._tabIndexedList.Remove(xmlElement);
				this._tabIndexedList.Add(m);
			}
			this.ArrangeTabIndex();
		}

		// Token: 0x060001C2 RID: 450 RVA: 0x00009100 File Offset: 0x00007300
		private void ArrangeTabIndex()
		{
			int num = 1;
			foreach (XmlElement xmlElement in this._tabIndexedList)
			{
				xmlElement["tabIndex"] = num.ToString();
				num++;
			}
			foreach (XmlElement xmlElement2 in this._sourceList)
			{
				if (!string.IsNullOrEmpty(xmlElement2.GetAttribute("tabIndex")))
				{
					xmlElement2["tabIndex"] = num.ToString();
					num++;
				}
			}
		}

		// Token: 0x060001C3 RID: 451 RVA: 0x000091E0 File Offset: 0x000073E0
		private int GetIndexCount()
		{
			return this._sourceList.Where<XmlElement>((XmlElement m) => !string.IsNullOrEmpty(m.GetAttribute("tabIndex"))).Count<XmlElement>();
		}

		// Token: 0x060001C4 RID: 452 RVA: 0x0000921C File Offset: 0x0000741C
		private void ReduceIndex()
		{
			int num = 1;
			foreach (XmlElement xmlElement in this._sourceList)
			{
				if (xmlElement.GetAttribute("tabIndex") != null)
				{
					xmlElement["tabIndex"] = num.ToString();
					num++;
				}
			}
		}

		// Token: 0x060001C5 RID: 453 RVA: 0x00009290 File Offset: 0x00007490
		private void iteratorAllNodes()
		{
			this._sourceList.Clear();
			this._tabIndexedList.Clear();
			this.iteratorAllNodes(this._mainRoot);
		}

		// Token: 0x060001C6 RID: 454 RVA: 0x000092B4 File Offset: 0x000074B4
		private void iteratorAllNodes(XmlElement container)
		{
			if (container.HasNodes)
			{
				foreach (XmlElement xmlElement in container.Nodes)
				{
					if (ComponentFactory.IsIncludeProperties(xmlElement.NodeName, "tabIndex"))
					{
						this._sourceList.Add(xmlElement);
					}
					this.iteratorAllNodes(xmlElement);
				}
			}
		}

		// Token: 0x060001C7 RID: 455 RVA: 0x00009328 File Offset: 0x00007528
		private void SortSourceList(bool autoIndex)
		{
			if (autoIndex)
			{
				int num = 1;
				foreach (XmlElement xmlElement in this._sourceList)
				{
					xmlElement["tabIndex"] = num.ToString();
					num++;
				}
			}
		}

		// Token: 0x060001C8 RID: 456 RVA: 0x000093B0 File Offset: 0x000075B0
		public bool Contains(XmlElement source)
		{
			IEnumerable<XmlElement> enumerable = this._sourceList.Where<XmlElement>((XmlElement s) => s.Name == source.Name);
			return enumerable.Count<XmlElement>() > 0;
		}

		// Token: 0x060001C9 RID: 457 RVA: 0x000093EC File Offset: 0x000075EC
		public void Show(bool isReOrder)
		{
			this.iteratorAllNodes();
			this.SortSourceList(isReOrder);
			foreach (XmlElement xmlElement in this._sourceList)
			{
				xmlElement.IsShowTabIndex = true;
			}
			this._isShown = true;
		}

		// Token: 0x060001CA RID: 458 RVA: 0x00009454 File Offset: 0x00007654
		public void Hide()
		{
			foreach (XmlElement xmlElement in this._sourceList)
			{
				xmlElement.IsShowTabIndex = false;
			}
			foreach (XmlElement xmlElement2 in this._tabIndexedList)
			{
				xmlElement2.IsShowTabIndex = false;
			}
			this._isShown = false;
		}

		// Token: 0x060001CB RID: 459 RVA: 0x000094F0 File Offset: 0x000076F0
		public void Sort()
		{
			this.SortSourceList(true);
		}

		// Token: 0x040000FC RID: 252
		private static Dictionary<PackageKey, ComponentTabIndexService> _services = new Dictionary<PackageKey, ComponentTabIndexService>();

		// Token: 0x040000FD RID: 253
		private bool _isShown;

		// Token: 0x040000FE RID: 254
		private List<XmlElement> _sourceList = new List<XmlElement>();

		// Token: 0x040000FF RID: 255
		private List<XmlElement> _tabIndexedList = new List<XmlElement>();

		// Token: 0x04000100 RID: 256
		private PackageKey _key;

		// Token: 0x04000101 RID: 257
		private XmlElement _mainRoot;
	}
}
