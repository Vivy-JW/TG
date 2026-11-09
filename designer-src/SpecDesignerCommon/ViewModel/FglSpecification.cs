using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Linq;
using System.Windows.Data;
using System.Xml.Linq;
using SpecDesignerCommon.Events;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000075 RID: 117
	public class FglSpecification
	{
		// Token: 0x17000132 RID: 306
		// (get) Token: 0x06000476 RID: 1142 RVA: 0x0001431E File Offset: 0x0001251E
		// (set) Token: 0x06000477 RID: 1143 RVA: 0x00014326 File Offset: 0x00012526
		private CodeSpecStatus Status { get; set; }

		// Token: 0x17000133 RID: 307
		// (get) Token: 0x06000478 RID: 1144 RVA: 0x0001432F File Offset: 0x0001252F
		// (set) Token: 0x06000479 RID: 1145 RVA: 0x00014337 File Offset: 0x00012537
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x17000134 RID: 308
		// (get) Token: 0x0600047A RID: 1146 RVA: 0x00014340 File Offset: 0x00012540
		// (set) Token: 0x0600047B RID: 1147 RVA: 0x00014348 File Offset: 0x00012548
		public XElement Source { get; private set; }

		// Token: 0x17000135 RID: 309
		// (get) Token: 0x0600047C RID: 1148 RVA: 0x00014354 File Offset: 0x00012554
		public string Ver
		{
			get
			{
				if (this.Source != null && this.Source.Attribute("ver") != null)
				{
					return this.Source.Attribute("ver").Value;
				}
				throw new ArgumentNullException("Ver");
			}
		}

		// Token: 0x17000136 RID: 310
		// (get) Token: 0x0600047D RID: 1149 RVA: 0x000143A5 File Offset: 0x000125A5
		public ObservableCollection<FglComponent> Components
		{
			get
			{
				return this._components;
			}
		}

		// Token: 0x17000137 RID: 311
		// (get) Token: 0x0600047E RID: 1150 RVA: 0x000143AD File Offset: 0x000125AD
		// (set) Token: 0x0600047F RID: 1151 RVA: 0x000143B5 File Offset: 0x000125B5
		public FglSpecContent SpecContent { get; private set; }

		// Token: 0x17000138 RID: 312
		// (get) Token: 0x06000480 RID: 1152 RVA: 0x000143BE File Offset: 0x000125BE
		// (set) Token: 0x06000481 RID: 1153 RVA: 0x000143C6 File Offset: 0x000125C6
		public CollectionViewSource FilteredComponents
		{
			get
			{
				return this._filteredComponents;
			}
			set
			{
				this._filteredComponents = value;
			}
		}

		// Token: 0x06000482 RID: 1154 RVA: 0x000143D0 File Offset: 0x000125D0
		public FglSpecification()
		{
			this.Status = CodeSpecStatus.NULL;
			this._filteredComponents = new CollectionViewSource();
			this._filteredComponents.Source = this._components;
			this._filteredComponents.Filter += this._filteredComponents_Filter;
		}

		// Token: 0x06000483 RID: 1155 RVA: 0x00014428 File Offset: 0x00012628
		private void _filteredComponents_Filter(object sender, FilterEventArgs e)
		{
			FglComponent fglComponent = e.Item as FglComponent;
			e.Accepted = (fglComponent.Status & CodeSpecStatus.DELETE) == CodeSpecStatus.NULL;
		}

		// Token: 0x06000484 RID: 1156 RVA: 0x00014454 File Offset: 0x00012654
		public void Add(FglComponent comp)
		{
			this._components.Add(comp);
			if ((this.Status & CodeSpecStatus.LOADED) == CodeSpecStatus.LOADED)
			{
				comp.Status |= CodeSpecStatus.MODIFY;
				EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeSpecChangedEvent>().Publish(this.ProgramKey);
			}
		}

		// Token: 0x06000485 RID: 1157 RVA: 0x000144A1 File Offset: 0x000126A1
		public void Remove(FglComponent comp)
		{
			comp.Status = CodeSpecStatus.DELETE;
			this._filteredComponents.View.Refresh();
			EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeSpecChangedEvent>().Publish(this.ProgramKey);
		}

		// Token: 0x06000486 RID: 1158 RVA: 0x000144F0 File Offset: 0x000126F0
		public XElement ToXElement()
		{
			XElement source = this.Source;
			source.RemoveNodes();
			IEnumerable<FglComponent> enumerable = this._components.Where<FglComponent>((FglComponent com) => (com.Status & CodeSpecStatus.DELETE) == CodeSpecStatus.DELETE);
			foreach (FglComponent fglComponent in enumerable)
			{
				source.Add(fglComponent.ToXElement());
			}
			IEnumerable<FglComponent> enumerable2 = this._components.Where<FglComponent>((FglComponent com) => (com.Status & CodeSpecStatus.DELETE) == CodeSpecStatus.NULL);
			foreach (FglComponent fglComponent2 in enumerable2)
			{
				source.Add(fglComponent2.ToXElement());
			}
			if (this.SpecContent != null)
			{
				source.Add(this.SpecContent.ToXElement());
			}
			return source;
		}

		// Token: 0x06000487 RID: 1159 RVA: 0x00014600 File Offset: 0x00012800
		public static FglSpecification Parse(string xml, TzpManager tzpManager)
		{
			if (string.IsNullOrEmpty(xml))
			{
				return null;
			}
			XElement xelement = XElement.Parse(xml, LoadOptions.None);
			FglSpecification fglSpecification = new FglSpecification();
			fglSpecification.ProgramKey = new PackageKey(tzpManager.ProgramName, tzpManager.Type);
			fglSpecification.Source = xelement;
			foreach (XElement xelement2 in xelement.Elements())
			{
				string localName = xelement2.Name.LocalName;
				string text;
				if ((text = localName) != null && text == "comp")
				{
					FglComponent fglComponent = FglComponent.Parse(fglSpecification, xelement2);
					fglSpecification.Add(fglComponent);
				}
			}
			fglSpecification.Status = CodeSpecStatus.LOADED;
			return fglSpecification;
		}

		// Token: 0x040001BE RID: 446
		private ObservableCollection<FglComponent> _components = new ObservableCollection<FglComponent>();

		// Token: 0x040001BF RID: 447
		private CollectionViewSource _filteredComponents;
	}
}
